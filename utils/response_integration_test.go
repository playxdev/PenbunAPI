//go:build integration

package utils

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_ResponseFormat(t *testing.T) {
	app := fiber.New()

	app.Get("/success", func(c *fiber.Ctx) error {
		return SuccessResponse(c, "ok", fiber.Map{"key": "value"})
	})
	app.Get("/fail", func(c *fiber.Ctx) error {
		return FailResponse(c, "invalid")
	})
	app.Get("/error", func(c *fiber.Ctx) error {
		return ErrorResponse(c, "server error")
	})

	t.Run("success format", func(t *testing.T) {
		resp, _ := app.Test(httptest.NewRequest("GET", "/success", nil))
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "success", body["status"])
		assert.Equal(t, "ok", body["message"])
	})

	t.Run("fail format", func(t *testing.T) {
		resp, _ := app.Test(httptest.NewRequest("GET", "/fail", nil))
		assert.Equal(t, 400, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "fail", body["status"])
	})

	t.Run("error format", func(t *testing.T) {
		resp, _ := app.Test(httptest.NewRequest("GET", "/error", nil))
		assert.Equal(t, 500, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "error", body["status"])
	})
}
