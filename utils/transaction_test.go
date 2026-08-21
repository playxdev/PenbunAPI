package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionStep_Struct(t *testing.T) {
	step := TransactionStep{
		Name:  "TestStep",
		Query: "SELECT * FROM tb_test",
		Args:  []interface{}{"arg1", 2},
	}

	assert.Equal(t, "TestStep", step.Name)
	assert.Equal(t, "SELECT * FROM tb_test", step.Query)
	assert.Len(t, step.Args, 2)
	assert.Equal(t, int64(0), step.RowsAffected)
}

func TestTransactionStep_DefaultValues(t *testing.T) {
	step := TransactionStep{}

	assert.Empty(t, step.Name)
	assert.Empty(t, step.Query)
	assert.Nil(t, step.Args)
	assert.Equal(t, int64(0), step.RowsAffected)
}
