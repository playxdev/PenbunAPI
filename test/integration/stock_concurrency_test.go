//go:build integration

package integration

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestStockConcurrency_AppLockSerializesSameStockKey(t *testing.T) {
	db := openTestDB(t)
	key := fmt.Sprintf("PENBUN:integration-stock:%d", time.Now().UnixNano())

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("begin first transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	acquireAppLock(t, tx, key)

	acquired := make(chan time.Duration, 1)
	go func() {
		secondTx, err := db.BeginTx(t.Context(), nil)
		if err != nil {
			return
		}
		defer secondTx.Rollback()
		started := time.Now()
		if _, err := secondTx.ExecContext(t.Context(), `
EXEC sp_getapplock @Resource = @p1, @LockMode = 'Exclusive',
    @LockOwner = 'Transaction', @LockTimeout = 5000`, key); err == nil {
			acquired <- time.Since(started)
		}
	}()

	time.Sleep(150 * time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit first transaction: %v", err)
	}

	select {
	case waited := <-acquired:
		if waited < 100*time.Millisecond {
			t.Fatalf("second stock lock waited %s; expected serialization", waited)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("second transaction did not acquire the stock lock")
	}
}

func acquireAppLock(t *testing.T, tx *sql.Tx, key string) {
	t.Helper()
	if _, err := tx.ExecContext(t.Context(), `
EXEC sp_getapplock @Resource = @p1, @LockMode = 'Exclusive',
    @LockOwner = 'Transaction', @LockTimeout = 5000`, key); err != nil {
		t.Fatalf("acquire application lock: %v", err)
	}
}
