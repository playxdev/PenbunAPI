package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTxLog_NilLogger(t *testing.T) {
	assert.NotPanics(t, func() {
		txLog("test message %s", "arg")
	})
}

func TestTransactionStep_Fields(t *testing.T) {
	step := TransactionStep{
		Name:         "TestStep",
		Query:        "INSERT INTO test VALUES (?)",
		Args:         []interface{}{"value1", 42},
		RowsAffected: 1,
	}

	assert.Equal(t, "TestStep", step.Name)
	assert.Equal(t, "INSERT INTO test VALUES (?)", step.Query)
	assert.Len(t, step.Args, 2)
	assert.Equal(t, int64(1), step.RowsAffected)
}

func TestTransactionStep_ZeroValue(t *testing.T) {
	var step TransactionStep

	assert.Equal(t, "", step.Name)
	assert.Equal(t, "", step.Query)
	assert.Nil(t, step.Args)
	assert.Equal(t, int64(0), step.RowsAffected)
}
