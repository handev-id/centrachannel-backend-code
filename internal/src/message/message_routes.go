package message

import "github.com/gofiber/fiber/v3"

func RegisterConversationRoutes(group fiber.Router, handler *MessageHandler) {
	group.Get("/:conversationId/messages", handler.List)
	group.Post("/:conversationId/messages", handler.Send)
}

func RegisterRoutes(group fiber.Router, handler *MessageHandler) {
	group.Put("/:id", handler.UpdateStatus)
}
