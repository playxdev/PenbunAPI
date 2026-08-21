package controllers

import "github.com/gofiber/fiber/v2"

func getUsername(c *fiber.Ctx) string {
	if u := c.Locals("username"); u != nil {
		if us, ok := u.(string); ok && us != "" {
			return us
		}
	}
	return "SYSTEM"
}
