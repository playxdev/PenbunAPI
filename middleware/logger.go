package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
)

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		status := c.Response().StatusCode()
		user := "-"
		if u := c.Locals("username"); u != nil {
			if us, ok := u.(string); ok && us != "" {
				user = us
			}
		}

		line := fmt.Sprintf("%s %s | status=%s user=%s",
			c.Method(), config.ShortPath(c.Path()), statusTag(status), user)

		switch {
		case status >= 500:
			config.LogError("%s", line)
		case status >= 400:
			config.LogWarn("%s", line)
		default:
			config.LogInfo("%s", line)
		}

		return err
	}
}

func statusTag(code int) string {
	switch {
	case code >= 500:
		return fmt.Sprintf("%d_ERROR", code)
	case code >= 400:
		return fmt.Sprintf("%d_WARN", code)
	case code >= 300:
		return fmt.Sprintf("%d_REDIRECT", code)
	default:
		return fmt.Sprintf("%d_OK", code)
	}
}
