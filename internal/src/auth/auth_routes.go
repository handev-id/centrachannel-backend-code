package auth

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(router fiber.Router, handler *AuthHandler) {
	auth := router.Group("/api/auth")
	auth.Post("/register", middleware.Tenant(handler.Register))
	auth.Post("/login", middleware.Tenant(handler.Login))
	auth.Get("/check-token", middleware.Tenant(handler.CheckToken))
	auth.Delete("/logout", handler.Logout)
}
