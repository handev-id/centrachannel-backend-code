package auth

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(router fiber.Router, handler *AuthHandler) {
	auth := router.Group("/api/auth")
	auth.Post("/register", handler.Register)
	auth.Post("/login", handler.Login)
	auth.Get("/check-token", handler.CheckToken)
	auth.Delete("/logout", handler.Logout)
}
