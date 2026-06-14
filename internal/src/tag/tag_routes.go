package tag

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *TagHandler) {
	group.Get("/", handler.List)
	group.Post("/", handler.Store)
	group.Put("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
}
