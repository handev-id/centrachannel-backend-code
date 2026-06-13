package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"

	"centrachannel/internal/di"
	"centrachannel/internal/utils/response"
)

func main() {
	// Initialize DI container (loads config, DB, logger)
	c, err := di.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer func() {
		if cerr := c.Close(); cerr != nil {
			log.Printf("Error closing container: %v", cerr)
		}
	}()

	cfg := c.Config
	logger := c.Logger
    // db := c.DB // retained for future use


	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "CentraChannel API v1.0.0",
	})

	// Middleware
	app.Use(middleware.LoggerMiddleware())
	app.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))

	// Health check route
	app.Get("/", func(c fiber.Ctx) error {
		return response.OK(c, "Welcome to CentraChannel API", fiber.Map{
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// Health check endpoint
	app.Get("/health", func(c fiber.Ctx) error {
		return response.OK(c, "Server is healthy", fiber.Map{
			"status": "ok",
		})
	})

	logger.Info("Starting server on port", cfg.Port)

	// Start server
	if err := app.Listen(":" + fmt.Sprintf("%d", cfg.Port)); err != nil {
		logger.Fatal("Server error", err)
	}
}