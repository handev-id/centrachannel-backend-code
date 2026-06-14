package note

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterConversationRoutes(group fiber.Router, handler *NoteHandler) {
	group.Get("/:conversationId/notes", middleware.Tenant(handler.List))
	group.Post("/:conversationId/notes", middleware.Tenant(handler.Store))
}

func RegisterRoutes(group fiber.Router, handler *NoteHandler) {
	group.Put("/:id", middleware.Tenant(handler.Update))
	group.Delete("/:id", middleware.Tenant(handler.Delete))
}
