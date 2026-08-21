package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestLogout_NoToken(t *testing.T) {
	app := fiber.New()
	app.Post("/api/v1/public/logout", Logout)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest("POST", "/api/v1/public/logout", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "fail", result["status"])
}

func TestLogin_InvalidBody(t *testing.T) {
	app := fiber.New()
	app.Post("/api/v1/public/login", Login)

	body := strings.NewReader(`invalid json`)
	req := httptest.NewRequest("POST", "/api/v1/public/login", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "fail", result["status"])
}

func TestLogin_MissingFields(t *testing.T) {
	app := fiber.New()
	app.Post("/api/v1/public/login", Login)

	body := strings.NewReader(`{"username": ""}`)
	req := httptest.NewRequest("POST", "/api/v1/public/login", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "fail", result["status"])
}
