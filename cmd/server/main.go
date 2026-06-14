package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/src/auth"
	"centrachannel/internal/src/user"
	"centrachannel/internal/utils/response"
)

func main() {
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

	app := fiber.New(fiber.Config{
		AppName: "CentraChannel API v1.0.0",
	})

	app.Use(middleware.TenantMiddleware(c.Redis, c.DB))
	app.Use(middleware.LoggerMiddleware())
	app.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))

	app.Get("/", func(c fiber.Ctx) error {
		return response.OK(c, "Welcome to CentraChannel API", fiber.Map{
			"version": "1.0.0",
			"status":  "running",
		})
	})

	authHandler := auth.NewAuthHandler(c)
	auth.RegisterRoutes(app, authHandler)

	userGroup := app.Group("/api/user", middleware.AuthMiddleware(cfg))
	userHandler := user.NewUserHandler(c)
	user.RegisterRoutesByGroup(userGroup, userHandler)

	c.Logger.Info("Starting server on port", cfg.Port)

	if err := app.Listen(":" + fmt.Sprintf("%d", cfg.Port)); err != nil {
		c.Logger.Fatal("Server error", err)
	}
}
