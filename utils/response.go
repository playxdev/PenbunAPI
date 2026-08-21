package utils

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
)

func SuccessResponse(c *fiber.Ctx, message string, data interface{}) error {
	config.LogInfo("%s %s | status=success msg=%q", c.Method(), config.ShortPath(c.Path()), message)
	return c.JSON(models.ApiResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func FailResponse(c *fiber.Ctx, message string) error {
	config.LogWarn("%s %s | status=fail msg=%q", c.Method(), config.ShortPath(c.Path()), message)
	return c.Status(400).JSON(models.ApiResponse{
		Status:  "fail",
		Message: message,
	})
}

func ErrorResponse(c *fiber.Ctx, message string) error {
	config.LogError("%s %s | status=error msg=%q", c.Method(), config.ShortPath(c.Path()), message)
	return c.Status(500).JSON(models.ApiResponse{
		Status:  "error",
		Message: message,
	})
}

func LoginSuccessResponse(c *fiber.Ctx, token, message string) error {
	config.LogInfo("%s %s | status=login_success msg=%q", c.Method(), config.ShortPath(c.Path()), message)
	return c.JSON(models.ApiResponse{
		Status:  "success",
		Token:   token,
		Message: message,
	})
}

func LoginFailResponse(c *fiber.Ctx, message string) error {
	config.LogWarn("%s %s | status=login_fail msg=%q", c.Method(), config.ShortPath(c.Path()), message)
	return c.Status(401).JSON(models.ApiResponse{
		Status:  "fail",
		Message: message,
	})
}

func UnauthorizedResponse(c *fiber.Ctx) error {
	config.LogWarn("%s %s | status=unauthorized", c.Method(), config.ShortPath(c.Path()))
	return c.Status(401).JSON(models.ApiResponse{
		Status:  "error",
		Message: "Unauthorized",
	})
}
