package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTMiddleware_MissingHeader(t *testing.T) {
	app := fiber.New()
	app.Use(JWTMiddleware("secret"))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/protected", nil))
	assert.Equal(t, 401, resp.StatusCode)
}

func TestJWTMiddleware_InvalidFormat(t *testing.T) {
	app := fiber.New()
	app.Use(JWTMiddleware("secret"))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	resp, _ := app.Test(req)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	app := fiber.New()
	app.Use(JWTMiddleware("my-secret"))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	claims := jwt.MapClaims{
		"username":   "testuser",
		"user_level": "ADMIN",
		"exp":        time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("my-secret"))
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	app := fiber.New()
	app.Use(JWTMiddleware("my-secret"))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	claims := jwt.MapClaims{
		"username":   "testuser",
		"user_level": "ADMIN",
		"exp":        time.Now().Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("my-secret"))
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, _ := app.Test(req)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestJWTMiddleware_WrongSecret(t *testing.T) {
	app := fiber.New()
	app.Use(JWTMiddleware("real-secret"))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	claims := jwt.MapClaims{
		"username": "testuser",
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("wrong-secret"))
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, _ := app.Test(req)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestJWTMiddleware_InvalidSignature(t *testing.T) {
	app := fiber.New()
	app.Use(JWTMiddleware("secret"))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	resp, _ := app.Test(req)
	assert.Equal(t, 401, resp.StatusCode)
}
