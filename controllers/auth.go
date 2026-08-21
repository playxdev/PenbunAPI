package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return utils.FailResponse(c, "Username and password are required")
	}

	var user models.User
	err := config.DB.QueryRow(
		"SELECT autoID, user_name, user_password, user_level FROM tb_users WHERE user_name = ? AND is_delete = 0",
		req.Username,
	).Scan(&user.AutoID, &user.UserName, &user.UserPassword, &user.UserLevel)
	if err != nil {
		config.TransactionLogger.Printf("LOGIN FAIL | user=%s | error=user not found", req.Username)
		return utils.LoginFailResponse(c, "Invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.UserPassword), []byte(req.Password)); err != nil {
		config.TransactionLogger.Printf("LOGIN FAIL | user=%s | error=invalid password", req.Username)
		return utils.LoginFailResponse(c, "Invalid username or password")
	}

	claims := jwt.MapClaims{
		"username":   user.UserName,
		"user_level": user.UserLevel,
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.Cfg.JWTSecret))
	if err != nil {
		config.TransactionLogger.Printf("LOGIN FAIL | user=%s | error=token generation failed", req.Username)
		return utils.ErrorResponse(c, "Failed to generate token")
	}

	config.TransactionLogger.Printf("LOGIN OK | user=%s level=%s", req.Username, user.UserLevel)
	return utils.LoginSuccessResponse(c, tokenStr, "Login successful")
}

func Logout(c *fiber.Ctx) error {
	tokenStr := c.Locals("token")
	if tokenStr == nil {
		return utils.FailResponse(c, "No token provided")
	}

	token, ok := tokenStr.(string)
	if !ok || token == "" {
		return utils.FailResponse(c, "Invalid token")
	}

	config.BlacklistToken(token)
	config.TransactionLogger.Printf("LOGOUT OK | token blacklisted")
	return c.JSON(models.ApiResponse{
		Status:  "success",
		Message: "Logged out successfully",
	})
}
