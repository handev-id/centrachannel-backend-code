package contact

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *ContactHandler) {
	group.Get("/", handler.List)
	group.Post("/", handler.Store)
	group.Get("/:id", handler.Show)
	group.Put("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
	group.Post("/:id/merge", handler.Merge)
	group.Post("/:id/unmerge", handler.Unmerge)
	group.Get("/:id/conversations", handler.Conversations)
}
