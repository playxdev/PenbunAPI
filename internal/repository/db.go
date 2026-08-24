// Path: internal/repository/db.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	mssql "github.com/microsoft/go-mssqldb"
)

// Timeout ต่อชนิดงาน — ตั้งจากลักษณะงาน ไม่ใช่ค่าเดียวทั้งระบบ
const (
	TimeoutLookup   = 5 * time.Second
	TimeoutRead     = 15 * time.Second
	TimeoutWrite    = 10 * time.Second
	TimeoutPostDoc  = 60 * time.Second
	deadlockRetries = 2
)

// Executor ครอบทั้ง *sql.DB และ *sql.Tx เพื่อให้ repository เขียนครั้งเดียว
// ใช้ได้ทั้งในและนอก transaction
type Executor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type DB struct {
	sql *sql.DB
	log *slog.Logger
}

func New(db *sql.DB, log *slog.Logger) *DB {
	return &DB{sql: db, log: log}
}

func (d *DB) Exec() Executor    { return d.sql }
func (d *DB) Raw() *sql.DB      { return d.sql }
func (d *DB) Log() *slog.Logger { return d.log }

func (d *DB) Ping(ctx context.Context) error { return d.sql.PingContext(ctx) }

// WithTx รัน fn ในหนึ่ง transaction
//
// รับเป็น closure ไม่ใช่ list ของคำสั่ง เพราะ fn ต้องอ่านผลลัพธ์ระหว่างทางได้
// เช่น สร้างเอกสาร header แล้วเอา autoID ที่เพิ่งเกิดไปใส่ items ต่อในทรานแซกชันเดียว
//
// deadlock (1205) จะ retry ให้อัตโนมัติ เพราะเป็น error ที่ผู้ใช้แก้อะไรไม่ได้
func (d *DB) WithTx(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	var lastErr error
	for attempt := 0; attempt <= deadlockRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt*50) * time.Millisecond):
			}
			d.log.Warn("retrying transaction after deadlock", "attempt", attempt)
		}
		err := d.runTx(ctx, fn)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isDeadlock(err) {
			return err
		}
	}
	return lastErr
}

func (d *DB) runTx(ctx context.Context, fn func(context.Context, *sql.Tx) error) (err error) {
	start := time.Now()
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			d.log.Error("transaction panic", "panic", r, "duration", time.Since(start))
			// re-panic ให้ recover middleware จัดการต่อ
			// การกลืน panic ไว้เงียบ ๆ ทำให้ bug หายไปจาก log
			panic(r)
		}
	}()

	if err = fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			d.log.Error("rollback failed", "error", rbErr)
		}
		d.log.Debug("transaction rolled back", "duration", time.Since(start), "error", err)
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	d.log.Debug("transaction committed", "duration", time.Since(start))
	return nil
}

// AppLock จองล็อกระดับ application ภายใน transaction ปัจจุบัน
//
// จำเป็นเพราะ USP_APPLY_STOCK_MOVEMENT ของ v7 เช็คสต็อกด้วย SELECT ที่ไม่มี
// UPDLOCK/HOLDLOCK ⇒ การโพสต์เอกสารสองใบพร้อมกันบน SKU/คลังเดียวกัน อ่าน
// qty_onhand ค่าเดิมทั้งคู่ ผ่านด่านทั้งคู่ แล้วสต็อกติดลบ
//
// ล็อกจะถูกปล่อยอัตโนมัติเมื่อ transaction จบ (LockOwner = 'Transaction')
func AppLock(ctx context.Context, tx *sql.Tx, resource string, waitMS int) error {
	const q = `
DECLARE @rc INT;
EXEC @rc = sp_getapplock
     @Resource = @p1, @LockMode = 'Exclusive',
     @LockOwner = 'Transaction', @LockTimeout = @p2;
IF @rc < 0
    RAISERROR (N'APPLOCK: ไม่สามารถจองล็อก %s ได้ (rc=%d)', 16, 1, @p1, @rc);`
	_, err := tx.ExecContext(ctx, q, resource, waitMS)
	return err
}

func isDeadlock(err error) bool {
	var me mssql.Error
	return errors.As(err, &me) && me.Number == 1205
}
