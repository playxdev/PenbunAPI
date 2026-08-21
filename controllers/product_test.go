package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateBusinessID_FirstID(t *testing.T) {
	result := generateBusinessID("PDT", 1)
	assert.Equal(t, "PDTA000001", result)
}

func TestGenerateBusinessID_SecondID(t *testing.T) {
	result := generateBusinessID("PDT", 2)
	assert.Equal(t, "PDTA000002", result)
}

func TestGenerateBusinessID_MaxSeries(t *testing.T) {
	result := generateBusinessID("PDT", 999999)
	assert.Equal(t, "PDTA999999", result)
}

func TestGenerateBusinessID_NextSeries(t *testing.T) {
	result := generateBusinessID("PDT", 1000000)
	assert.Equal(t, "PDTB000001", result)
}

func TestGenerateBusinessID_ThirdSeries(t *testing.T) {
	result := generateBusinessID("PDT", 2000001)
	assert.Equal(t, "PDTC000003", result)
}

func TestGenerateBusinessID_DifferentPrefix(t *testing.T) {
	result := generateBusinessID("CUS", 1)
	assert.Equal(t, "CUSA000001", result)
}

func TestGenerateBusinessID_LargeAutoID(t *testing.T) {
	result := generateBusinessID("PDT", 26000000)
	assert.Equal(t, "PDTA000026", result)
}

func TestGenerateBusinessID_WrapsAround(t *testing.T) {
	result := generateBusinessID("PDT", 26000001)
	assert.Equal(t, "PDTA000027", result)
}

func TestGenerateBusinessID_ZeroAutoID(t *testing.T) {
	result := generateBusinessID("PDT", 0)
	assert.Equal(t, "PDTA000000", result)
}
