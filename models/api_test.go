package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiResponse_Success(t *testing.T) {
	resp := ApiResponse{
		Status:  "success",
		Message: "Operation completed",
		Data:    map[string]string{"key": "value"},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded ApiResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "success", decoded.Status)
	assert.Equal(t, "Operation completed", decoded.Message)
}

func TestApiResponse_Error(t *testing.T) {
	resp := ApiResponse{
		Status:  "error",
		Message: "Something went wrong",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded ApiResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "error", decoded.Status)
	assert.Equal(t, "Something went wrong", decoded.Message)
	assert.Nil(t, decoded.Data)
}

func TestApiResponse_Login(t *testing.T) {
	resp := ApiResponse{
		Status:  "success",
		Token:   "jwt-token-string",
		Message: "Login successful",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "success", decoded["status"])
	assert.Equal(t, "jwt-token-string", decoded["token"])
}

func TestLoginRequest(t *testing.T) {
	req := LoginRequest{
		Username: "admin",
		Password: "secret123",
	}

	assert.Equal(t, "admin", req.Username)
	assert.Equal(t, "secret123", req.Password)
}

func TestUser_HidesPassword(t *testing.T) {
	user := User{
		AutoID:       1,
		UserName:     "admin",
		UserPassword: "hashed-secret",
		UserLevel:    "ADMIN",
	}

	data, err := json.Marshal(user)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, float64(1), decoded["auto_id"])
	assert.Equal(t, "admin", decoded["user_name"])
	assert.Equal(t, "ADMIN", decoded["user_level"])
	_, passwordExposed := decoded["user_password"]
	assert.False(t, passwordExposed, "password should not be exposed in JSON")
}
