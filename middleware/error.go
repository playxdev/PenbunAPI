package middleware

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
)

func GlobalErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	config.LogError("%s %s | status=%d error=%s",
		c.Method(), config.ShortPath(c.Path()), code, err.Error())

	return c.Status(code).JSON(models.ApiResponse{
		Status:  "error",
		Message: err.Error(),
	})
}
