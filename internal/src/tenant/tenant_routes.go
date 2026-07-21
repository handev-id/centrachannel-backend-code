package tenant

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *TenantHandler) {
	group.Get("/", handler.Get)
	group.Put("/", handler.Update)
}
