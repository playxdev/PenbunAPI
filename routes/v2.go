package routes

import (
	"github.com/gofiber/fiber/v2"
)

func SetupV2Routes(app *fiber.App) {
	api := app.Group("/api/v2")

	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "PenbunAPI v2 endpoints coming soon",
		})
	})
}
