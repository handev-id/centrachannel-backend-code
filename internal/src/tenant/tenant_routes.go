package tenant

import "github.com/gofiber/fiber/v3"

func RegisterProtectedRoutes(group fiber.Router, handler *TenantHandler) {
	group.Get("/", handler.List)
	group.Get("/:id", handler.Show)
}
