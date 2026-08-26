package auth

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
)

func RegisterRoutes(app fiber.Router, prefix string, c *di.Container, middlewares ...any) {
	group := app.Group(prefix, middlewares...)
	handler := NewAuthHandler(c)
	group.Post("/register", middleware.Tenant(handler.Register))
	group.Post("/login", middleware.Tenant(handler.Login))
	group.Get("/check-token", middleware.Tenant(handler.CheckToken))
	group.Delete("/logout", handler.Logout)
}
