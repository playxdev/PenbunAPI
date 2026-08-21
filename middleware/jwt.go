package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"PenbunAPI/config"
	"PenbunAPI/utils"
)

func JWTMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			config.LogWarn("%s %s | reason=missing_auth_header", c.Method(), config.ShortPath(c.Path()))
			return utils.UnauthorizedResponse(c)
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			config.LogWarn("%s %s | reason=malformed_auth_header", c.Method(), config.ShortPath(c.Path()))
			return utils.UnauthorizedResponse(c)
		}

		tokenStr := parts[1]

		if config.IsTokenBlacklisted(tokenStr) {
			config.LogWarn("%s %s | reason=token_blacklisted", c.Method(), config.ShortPath(c.Path()))
			return utils.UnauthorizedResponse(c)
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			config.LogWarn("%s %s | reason=invalid_token error=%v", c.Method(), config.ShortPath(c.Path()), err)
			return utils.UnauthorizedResponse(c)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			config.LogWarn("%s %s | reason=invalid_claims", c.Method(), config.ShortPath(c.Path()))
			return utils.UnauthorizedResponse(c)
		}

		username := claims["username"]
		userLevel := claims["user_level"]
		c.Locals("username", username)
		c.Locals("user_level", userLevel)
		c.Locals("token", tokenStr)

		config.LogInfo("%s %s | status=authenticated user=%v level=%v",
			c.Method(), config.ShortPath(c.Path()), username, userLevel)

		return c.Next()
	}
}
