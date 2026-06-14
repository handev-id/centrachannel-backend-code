package message

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterConversationRoutes(group fiber.Router, handler *MessageHandler) {
	group.Get("/:conversationId/messages", handler.List)
	group.Post("/:conversationId/messages", middleware.Tenant(handler.Send))
}

func RegisterRoutes(group fiber.Router, handler *MessageHandler) {
	group.Put("/:id", handler.UpdateStatus)
}
