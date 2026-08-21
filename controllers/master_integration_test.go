//go:build integration

package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"PenbunAPI/config"
	"PenbunAPI/middleware"
)

func initIntegrationDB() {
	if config.DB != nil {
		return
	}

	cfg := config.LoadEnv()
	config.ConnectDB(cfg)
	config.InitLogger(cfg)
}

func TestIntegration_InsertAndSelectCustomer(t *testing.T) {
	initIntegrationDB()

	app := fiber.New()
	jwtMW := middleware.JWTMiddleware(config.Cfg.JWTSecret)
	api := app.Group("/api/v1/protected", jwtMW)
	api.Post("/customer/insert", InsertCustomer)
	api.Get("/customer/all", SelectAllCustomer)

	token := generateIntegrationToken()

	body := strings.NewReader(`{"customer_name":"INTEGRATION TEST CUSTOMER","update_by":"TESTER"}`)
	req := httptest.NewRequest("POST", "/api/v1/protected/customer/insert", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestIntegration_InsertAndSelectVendor(t *testing.T) {
	initIntegrationDB()

	app := fiber.New()
	jwtMW := middleware.JWTMiddleware(config.Cfg.JWTSecret)
	api := app.Group("/api/v1/protected", jwtMW)
	api.Post("/vendor/insert", InsertVendor)
	api.Get("/vendor/all", SelectAllVendor)

	token := generateIntegrationToken()

	body := strings.NewReader(`{"vendor_name":"INTEGRATION TEST VENDOR","update_by":"TESTER"}`)
	req := httptest.NewRequest("POST", "/api/v1/protected/vendor/insert", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestIntegration_InsertAndSelectBook(t *testing.T) {
	initIntegrationDB()

	app := fiber.New()
	jwtMW := middleware.JWTMiddleware(config.Cfg.JWTSecret)
	api := app.Group("/api/v1/protected", jwtMW)
	api.Post("/book/insert", InsertBook)
	api.Get("/book/all", SelectAllBook)

	token := generateIntegrationToken()

	body := strings.NewReader(`{"book_name":"INTEGRATION TEST BOOK","update_by":"TESTER"}`)
	req := httptest.NewRequest("POST", "/api/v1/protected/book/insert", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func generateIntegrationToken() string {
	claims := jwt.MapClaims{
		"username":   "tester",
		"user_level": "ADMIN",
		"exp":        time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(config.Cfg.JWTSecret))
	return tokenStr
}
