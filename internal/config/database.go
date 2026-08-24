// Path: internal/config/database.go
package config

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	// driver name "sqlserver" — placeholder เป็น @p1 / sql.Named
	// ห้ามใช้ชื่อ "mssql" (legacy, placeholder เป็น ?) เพราะ query ทั้ง repo
	// เขียนด้วย @pN ถ้าสลับชื่อ driver จะ compile ผ่านแต่ runtime พังทุกเส้น
	_ "github.com/microsoft/go-mssqldb"
)

func OpenDB(ctx context.Context, c *Config) (*sql.DB, error) {
	q := url.Values{}
	q.Set("database", c.DBName)
	q.Set("encrypt", boolStr(c.DBEncrypt))
	q.Set("TrustServerCertificate", boolStr(c.DBTrustCert))
	q.Set("app name", "PenbunAPI/"+c.Version)
	q.Set("connection timeout", "10")

	dsn := (&url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     fmt.Sprintf("%s:%s", c.DBHost, c.DBPort),
		RawQuery: q.Encode(),
	}).String()

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(c.DBMaxOpen)
	db.SetMaxIdleConns(c.DBMaxIdle)
	db.SetConnMaxLifetime(c.DBConnMaxTTL)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
