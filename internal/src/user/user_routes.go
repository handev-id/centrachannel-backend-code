package user

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(router fiber.Router, handler *UserHandler) {
	users := router.Group("/api/user")
	RegisterRoutesByGroup(users, handler)
}

func RegisterRoutesByGroup(group fiber.Router, handler *UserHandler) {
	group.Get("/", handler.List)
	group.Post("/", handler.Store)
	group.Get("/:id", handler.Show)
	group.Put("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
}
