package config

import (
	"os"
	"testing"
)

func TestLoadEnv_Defaults(t *testing.T) {
	os.Clearenv()

	cfg := LoadEnv()

	if cfg.DBHost != "localhost" {
		t.Errorf("expected localhost, got %s", cfg.DBHost)
	}
	if cfg.DBPort != "1433" {
		t.Errorf("expected 1433, got %s", cfg.DBPort)
	}
	if cfg.FiberPort != "8089" {
		t.Errorf("expected 8089, got %s", cfg.FiberPort)
	}
	if cfg.JWTSecret != "default-secret" {
		t.Errorf("expected default-secret, got %s", cfg.JWTSecret)
	}
	if cfg.LogFile != "logs/transaction.log" {
		t.Errorf("expected logs/transaction.log, got %s", cfg.LogFile)
	}
}

func TestLoadEnv_Override(t *testing.T) {
	os.Clearenv()
	os.Setenv("DB_HOST", "test-host")
	os.Setenv("DB_PORT", "9999")
	os.Setenv("FIBER_PORT", "3000")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("LOG_FILE", "/tmp/test.log")

	cfg := LoadEnv()

	if cfg.DBHost != "test-host" {
		t.Errorf("expected test-host, got %s", cfg.DBHost)
	}
	if cfg.DBPort != "9999" {
		t.Errorf("expected 9999, got %s", cfg.DBPort)
	}
	if cfg.FiberPort != "3000" {
		t.Errorf("expected 3000, got %s", cfg.FiberPort)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("expected test-secret, got %s", cfg.JWTSecret)
	}
	if cfg.LogFile != "/tmp/test.log" {
		t.Errorf("expected /tmp/test.log, got %s", cfg.LogFile)
	}

	os.Clearenv()
}

func TestLoadEnv_PartialOverride(t *testing.T) {
	os.Clearenv()
	os.Setenv("DB_HOST", "partial-host")

	cfg := LoadEnv()

	if cfg.DBHost != "partial-host" {
		t.Errorf("expected partial-host, got %s", cfg.DBHost)
	}
	if cfg.DBPort != "1433" {
		t.Errorf("expected default 1433, got %s", cfg.DBPort)
	}
	if cfg.FiberPort != "8089" {
		t.Errorf("expected default 8089, got %s", cfg.FiberPort)
	}

	os.Clearenv()
}
