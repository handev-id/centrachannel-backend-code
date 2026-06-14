package user

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(router fiber.Router, handler *UserHandler) {
	users := router.Group("/api/user")
	RegisterRoutesByGroup(users, handler)
}

func RegisterRoutesByGroup(group fiber.Router, handler *UserHandler) {
	group.Get("/", middleware.Tenant(handler.List))
	group.Post("/", middleware.Tenant(handler.Store))
	group.Get("/:id", middleware.Tenant(handler.Show))
	group.Put("/:id", middleware.Tenant(handler.Update))
	group.Delete("/:id", middleware.Tenant(handler.Delete))
}
