package note

import "github.com/gofiber/fiber/v3"

func RegisterConversationRoutes(group fiber.Router, handler *NoteHandler) {
	group.Get("/:conversationId/notes", handler.List)
	group.Post("/:conversationId/notes", handler.Store)
}

func RegisterRoutes(group fiber.Router, handler *NoteHandler) {
	group.Put("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
}
