// Path: internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config เก็บค่าตั้งค่าทั้งหมดของ process
// ค่าที่ไม่มี default = บังคับต้องตั้ง มิฉะนั้น Load() คืน error และ process ต้องตาย
type Config struct {
	// Database
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBEncrypt    bool
	DBTrustCert  bool
	DBMaxOpen    int
	DBMaxIdle    int
	DBConnMaxTTL time.Duration

	// HTTP
	Port         string
	CORSOrigins  []string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	BodyLimit    int

	// Auth
	JWTSecret     string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	AuthMaxFail   int
	BcryptCost    int
	AppLockWaitMS int

	// Ops
	LogLevel string
	AppEnv   string
	Version  string
}

func Load() (*Config, error) {
	c := &Config{
		DBHost:       str("DB_HOST", "localhost"),
		DBPort:       str("DB_PORT", "1433"),
		DBUser:       str("DB_USER", "sa"),
		DBPassword:   str("DB_PASSWORD", ""),
		DBName:       str("DB_NAME", "PENBUN"),
		DBEncrypt:    boolean("DB_ENCRYPT", true),
		DBTrustCert:  boolean("DB_TRUST_CERT", false),
		DBMaxOpen:    integer("DB_MAX_OPEN", 25),
		DBMaxIdle:    integer("DB_MAX_IDLE", 10),
		DBConnMaxTTL: duration("DB_CONN_MAX_LIFETIME", 30*time.Minute),

		Port:        str("FIBER_PORT", "8089"),
		CORSOrigins: list("CORS_ORIGINS"),
		ReadTimeout: duration("HTTP_READ_TIMEOUT", 30*time.Second),
		// WriteTimeout ต้องมากกว่า timeout ที่ยาวที่สุดของงาน (POST เอกสาร 60s)
		WriteTimeout: duration("HTTP_WRITE_TIMEOUT", 90*time.Second),
		IdleTimeout:  duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		BodyLimit:    integer("HTTP_BODY_LIMIT", 20*1024*1024),

		JWTSecret:     str("JWT_SECRET", ""),
		AccessTTL:     duration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:    duration("JWT_REFRESH_TTL", 168*time.Hour),
		AuthMaxFail:   integer("AUTH_MAX_FAIL", 5),
		BcryptCost:    integer("BCRYPT_COST", 10),
		AppLockWaitMS: integer("APPLOCK_WAIT_MS", 10000),

		LogLevel: str("LOG_LEVEL", "info"),
		AppEnv:   str("APP_ENV", "development"),
		Version:  str("APP_VERSION", "4.0.0"),
	}

	// ไม่มี default ให้ JWT_SECRET โดยเจตนา
	// การมี default ให้ secret แปลว่าใครก็ปลอม token ได้ถ้าลืมตั้ง env
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if len(c.CORSOrigins) == 0 {
		return nil, fmt.Errorf("CORS_ORIGINS is required (comma separated, no wildcard)")
	}
	for _, o := range c.CORSOrigins {
		if o == "*" {
			return nil, fmt.Errorf("CORS_ORIGINS must not contain '*' — API uses bearer tokens")
		}
	}
	if c.BcryptCost > 12 {
		return nil, fmt.Errorf("BCRYPT_COST above 12 makes every login exponentially slower")
	}
	return c, nil
}

func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

func str(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func integer(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func boolean(k string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func duration(k string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func list(k string) []string {
	raw := os.Getenv(k)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
