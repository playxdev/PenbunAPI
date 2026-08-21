package routes

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/controllers"
	"PenbunAPI/middleware"
)

func SetupPublicRoutes(app *fiber.App, jwtSecret string) {
	api := app.Group("/api/v1/public")

	api.Post("/login", controllers.Login)
	api.Post("/logout", middleware.JWTMiddleware(jwtSecret), controllers.Logout)
}
