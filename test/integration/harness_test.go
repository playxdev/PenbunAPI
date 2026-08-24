//go:build integration

package integration

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("PENBUN_INTEGRATION_DB_DSN")
	if dsn == "" {
		t.Skip("set PENBUN_INTEGRATION_DB_DSN to run database integration tests")
	}

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.PingContext(t.Context()); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	return db
}
