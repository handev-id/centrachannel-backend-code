package message

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
)

func RegisterConversationRoutes(app fiber.Router, handler *MessageHandler) {
	app.Get("/:conversationId/messages", middleware.Tenant(handler.List))
	app.Post("/:conversationId/messages", middleware.Tenant(handler.Send))
}

func RegisterRoutes(app fiber.Router, prefix string, c *di.Container, middlewares ...any) {
	group := app.Group(prefix, middlewares...)
	handler := NewMessageHandler(c)
	group.Get("/", middleware.Tenant(handler.List))
	group.Post("/", middleware.Tenant(handler.Send))
	group.Put("/:id", middleware.Tenant(handler.UpdateStatus))
}
