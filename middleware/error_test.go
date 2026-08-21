package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGlobalErrorHandler_FiberError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: GlobalErrorHandler,
	})
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/error", nil))
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "not found", result["message"])
}

func TestGlobalErrorHandler_GenericError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: GlobalErrorHandler,
	})
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.ErrInternalServerError
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/error", nil))
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "error", result["status"])
}

func TestGlobalErrorHandler_BadRequest(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: GlobalErrorHandler,
	})
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad request")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/error", nil))
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "bad request", result["message"])
}
