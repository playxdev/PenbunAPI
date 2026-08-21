package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func setupTestApp() *fiber.App {
	return fiber.New()
}

func TestSuccessResponse(t *testing.T) {
	app := setupTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return SuccessResponse(c, "operation successful", fiber.Map{"id": "123"})
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 200, resp.StatusCode)
}

func TestFailResponse(t *testing.T) {
	app := setupTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return FailResponse(c, "invalid input")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 400, resp.StatusCode)
}

func TestErrorResponse(t *testing.T) {
	app := setupTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return ErrorResponse(c, "internal error")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 500, resp.StatusCode)
}

func TestLoginSuccessResponse(t *testing.T) {
	app := setupTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return LoginSuccessResponse(c, "jwt-token", "login ok")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 200, resp.StatusCode)
}

func TestLoginFailResponse(t *testing.T) {
	app := setupTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return LoginFailResponse(c, "bad credentials")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 401, resp.StatusCode)
}

func TestUnauthorizedResponse(t *testing.T) {
	app := setupTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return UnauthorizedResponse(c)
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 401, resp.StatusCode)
}
