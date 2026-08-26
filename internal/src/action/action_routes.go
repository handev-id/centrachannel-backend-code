package action

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
)

func RegisterRoutes(app fiber.Router, prefix string, c *di.Container, middlewares ...any) {
	group := app.Group(prefix, middlewares...)
	handler := NewActionHandler(c)
	group.Put("/conversations/:id/read", middleware.Tenant(handler.MarkConversationRead))
}
