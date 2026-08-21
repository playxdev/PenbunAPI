package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"PenbunAPI/config"
	"PenbunAPI/middleware"
	"PenbunAPI/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config.LoadEnv()
	config.ConnectDB(config.Cfg)
	config.InitLogger(config.Cfg)

	app := fiber.New(fiber.Config{
		ServerHeader:          "PENBUN Powered by Fiber",
		AppName:               "API v3.2.0",
		Prefork:               false,
		CaseSensitive:         true,
		StrictRouting:         true,
		EnablePrintRoutes:     true,
		DisableStartupMessage: false,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
		BodyLimit:             20 * 1024 * 1024,
		ErrorHandler:          middleware.GlobalErrorHandler,
	})

	app.Use(recover.New())
	app.Use(middleware.RequestLogger())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Authorization,Content-Type",
	}))

	routes.SetupPublicRoutes(app, config.Cfg.JWTSecret)
	routes.SetupV1Routes(app, config.Cfg.JWTSecret)
	routes.SetupV2Routes(app)

	fmt.Println("========== Registered Routes ==========")
	for _, r := range app.GetRoutes() {
		if r.Method != "" && r.Path != "" {
			fmt.Printf("  %-6s %s\n", r.Method, r.Path)
		}
	}
	fmt.Println("======================================")

	go func() {
		addr := ":" + config.Cfg.FiberPort
		log.Printf("PenbunAPI server starting on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	if config.LogFile != nil {
		config.LogFile.Close()
	}

	log.Println("Server exited")
}
