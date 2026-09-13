// Path: internal/domain/auth/repo.go
package auth

import (
	"context"
	"database/sql"
	"errors"

	"penbun/api/internal/repository"
)

// User สะท้อนคอลัมน์ของ tb_users ที่การเข้าสู่ระบบต้องใช้
// ฟิลด์ 4 ตัวท้ายรองรับ Authentication Spec M001
type User struct {
	AutoID       int
	UserID       string
	UserName     string
	PasswordHash string
	UserLevel    string
	FullName     sql.NullString
	Email        sql.NullString

	FailCount    int
	Locked       bool
	MustChangePW bool
	LastLogin    sql.NullTime
}

var ErrUserNotFound = errors.New("user not found")

type Repo struct{ db *repository.DB }

func NewRepo(db *repository.DB) *Repo { return &Repo{db: db} }

const userColumns = `autoID, user_id, user_name, user_password, user_level,
	full_name, email, counting_password_fail, status_user_locked,
	status_change_pw, last_login_date`

func (r *Repo) ByUsername(ctx context.Context, username string) (*User, error) {
	const q = `SELECT TOP 1 ` + userColumns + `
	             FROM dbo.tb_users
	            WHERE user_name = @p1 AND is_delete = 0 AND is_active = 1`
	return scanUser(r.db.Exec().QueryRowContext(ctx, q, username))
}

func (r *Repo) ByUserID(ctx context.Context, userID string) (*User, error) {
	const q = `SELECT TOP 1 ` + userColumns + `
	             FROM dbo.tb_users
	            WHERE user_id = @p1 AND is_delete = 0 AND is_active = 1`
	return scanUser(r.db.Exec().QueryRowContext(ctx, q, userID))
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.AutoID, &u.UserID, &u.UserName, &u.PasswordHash, &u.UserLevel,
		&u.FullName, &u.Email, &u.FailCount, &u.Locked, &u.MustChangePW, &u.LastLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// RegisterFailure เพิ่มตัวนับรหัสผิด และล็อกบัญชีเมื่อถึงเพดาน
// ทำใน UPDATE คำสั่งเดียวเพื่อไม่ให้ผู้โจมตีที่ยิงพร้อมกันหลายเส้นเลี่ยงตัวนับได้
//
// ถ้าแถวปลดล็อกอยู่แต่ตัวนับยังค้างเต็มเพดาน แปลว่ามีคนแก้ status_user_locked
// ในฐานตรง ๆ โดยไม่ล้าง counting_password_fail กรณีนี้ถือว่าเริ่มนับรอบใหม่
// (ตั้งตัวนับเป็น 1 และคงสถานะปลดล็อก) ไม่ใช่ล็อกซ้ำทันทีที่พิมพ์ผิดครั้งเดียว
//
// คืนค่าสถานะล็อกหลังอัปเดตจริง service จะได้ไม่ต้องเดาจาก FailCount ที่อ่านมาก่อนหน้า
func (r *Repo) RegisterFailure(ctx context.Context, autoID, maxFail int) (bool, error) {
	// ต้อง OUTPUT ... INTO เพราะ tb_users มี AFTER UPDATE trigger อยู่
	// SQL Server ไม่ยอมให้ OUTPUT คืนค่าตรงบนตารางที่มี trigger เปิดใช้งาน
	const q = `
DECLARE @out TABLE (locked bit);
UPDATE dbo.tb_users
   SET counting_password_fail = CASE WHEN status_user_locked = 0 AND counting_password_fail >= @p2
                                     THEN 1
                                     ELSE counting_password_fail + 1 END,
       status_user_locked     = CASE WHEN status_user_locked = 0 AND counting_password_fail >= @p2
                                     THEN 0
                                     WHEN counting_password_fail + 1 >= @p2
                                     THEN 1
                                     ELSE status_user_locked END,
       update_by = N'System'
 OUTPUT inserted.status_user_locked INTO @out
 WHERE autoID = @p1;
SELECT TOP 1 locked FROM @out;`
	var locked bool
	err := r.db.Exec().QueryRowContext(ctx, q, autoID, maxFail).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrUserNotFound
	}
	if err != nil {
		return false, err
	}
	return locked, nil
}

func (r *Repo) RegisterSuccess(ctx context.Context, autoID int) error {
	const q = `
UPDATE dbo.tb_users
   SET counting_password_fail = 0,
       last_login_date = CAST(SYSDATETIMEOFFSET() AT TIME ZONE 'SE Asia Standard Time' AS DATETIME),
       update_by = user_name
 WHERE autoID = @p1`
	_, err := r.db.Exec().ExecContext(ctx, q, autoID)
	return err
}

func (r *Repo) SetPassword(ctx context.Context, autoID int, hash, updateBy string) error {
	const q = `
UPDATE dbo.tb_users
   SET user_password = @p2,
       status_change_pw = 0,
       counting_password_fail = 0,
       update_by = @p3
 WHERE autoID = @p1`
	_, err := r.db.Exec().ExecContext(ctx, q, autoID, hash, updateBy)
	return err
}

func (r *Repo) Unlock(ctx context.Context, userID, updateBy string) (int64, error) {
	const q = `
UPDATE dbo.tb_users
   SET status_user_locked = 0, counting_password_fail = 0, update_by = @p2
 WHERE user_id = @p1 AND is_delete = 0`
	res, err := r.db.Exec().ExecContext(ctx, q, userID, updateBy)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Privilege คือสิ่งที่ผู้ใช้คนหนึ่งทำได้กับ resource หนึ่งตัว ตามที่ฐานข้อมูลบอก
// ชื่อสี่ช่องตรงกับคอลัมน์ของ tb_privilege ซึ่งตรงกับช่องติ๊กสี่ช่องบนหน้าจอ
// M002_P0004 ของ Authentication Spec
type Privilege struct {
	Resource string
	View     bool
	Insert   bool
	Update   bool
	Delete   bool
}

// PrivilegesFor คืนสิทธิ์ผลลัพธ์ของผู้ใช้หนึ่งคน
//
// ทางหลักคือ vw_user_privilege ซึ่งรวมสิทธิ์ข้ามบทบาทแบบ union ให้แล้ว
// (ผู้ใช้ถือได้หลายบทบาท บทบาทใดให้ผ่านก็ผ่าน)
//
// ทางรองมีไว้เพราะ v12 เพิ่งวางตาราง RBAC ลงฐาน แต่ยังไม่มีอะไรผูกบทบาทให้
// ผู้ใช้ที่สร้างผ่าน POST /users — มีแต่ผู้ใช้ตั้งต้นที่ SEED ผูกไว้ให้ ผู้ใช้ที่
// ยังไม่มีแถวใน tb_user_role จึงตกมาที่บทบาทที่ role_code ตรงกับ user_level ของตัวเอง
// ซึ่ง SEED ตั้งไว้ให้เท่ากับกฎที่ API บังคับอยู่จริงพอดี
//
// ทางรองไม่ได้เขียนกฎซ้ำไว้ใน Go — มันอ่าน tb_privilege เหมือนกัน ต่างแค่ทางที่
// เดินไปถึงบทบาท ฐานข้อมูลจึงยังเป็นแหล่งความจริงแหล่งเดียว
// ตัดทางรองนี้ทิ้งได้เมื่อ POST /users เขียน tb_user_role ให้ทุกคนแล้ว
func (r *Repo) PrivilegesFor(ctx context.Context, userID, userLevel string) ([]Privilege, error) {
	const q = `
SELECT resource_code, can_view, can_insert, can_update, can_delete
  FROM dbo.vw_user_privilege
 WHERE user_id = @p1
UNION ALL
SELECT p.resource_code,
       CAST(p.can_view AS INT), CAST(p.can_insert AS INT),
       CAST(p.can_update AS INT), CAST(p.can_delete AS INT)
  FROM dbo.tb_privilege p
  INNER JOIN dbo.tb_role r ON r.autoID = p.ref_role_auto
 WHERE r.role_code = @p2
   AND r.is_delete = 0 AND r.is_active = 1
   AND p.is_delete = 0 AND p.is_active = 1
   AND NOT EXISTS (SELECT 1 FROM dbo.vw_user_privilege WHERE user_id = @p1)`

	rows, err := r.db.Exec().QueryContext(ctx, q, userID, userLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Privilege, 0, 32)
	for rows.Next() {
		var p Privilege
		var v, i, u, d int
		if err := rows.Scan(&p.Resource, &v, &i, &u, &d); err != nil {
			return nil, err
		}
		p.View, p.Insert, p.Update, p.Delete = v == 1, i == 1, u == 1, d == 1
		out = append(out, p)
	}
	return out, rows.Err()
}
