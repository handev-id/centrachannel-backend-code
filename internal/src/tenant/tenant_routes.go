package tenant

import "github.com/gofiber/fiber/v3"

func RegisterPublicRoutes(router fiber.Router, handler *TenantHandler) {
	router.Post("/api/tenants/onboard", handler.Onboard)
}

func RegisterProtectedRoutes(group fiber.Router, handler *TenantHandler) {
	group.Get("/", handler.List)
	group.Get("/:id", handler.Show)
}
