package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := &EnvConfig{
		LogFile: tmpFile.Name(),
	}

	InitLogger(cfg)

	assert.NotNil(t, LogFile)
	assert.NotNil(t, TransactionLogger)

	LogFile.Close()
}

func TestInitLogger_CreatesFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &EnvConfig{
		LogFile: tmpDir + "/new.log",
	}

	InitLogger(cfg)
	assert.NotNil(t, LogFile)
	assert.NotNil(t, TransactionLogger)

	LogFile.Close()
}
