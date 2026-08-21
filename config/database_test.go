package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectDB_ConnectionString(t *testing.T) {
	cfg := &EnvConfig{
		DBHost:     "myhost",
		DBPort:     "1433",
		DBUser:     "sa",
		DBPassword: "pass",
		DBName:     "testdb",
	}

	assert.Equal(t, "myhost", cfg.DBHost)
	assert.Equal(t, "1433", cfg.DBPort)
	assert.Equal(t, "sa", cfg.DBUser)
	assert.Equal(t, "pass", cfg.DBPassword)
	assert.Equal(t, "testdb", cfg.DBName)
}
