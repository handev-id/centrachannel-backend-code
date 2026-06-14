package profile

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *ProfileHandler) {
	group.Get("/", handler.List)
	group.Get("/:id", handler.Show)
	group.Put("/:id", handler.Update)
}
