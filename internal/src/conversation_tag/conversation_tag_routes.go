package conversation_tag

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
)

func RegisterRoutes(app fiber.Router, prefix string, c *di.Container, middlewares ...any) {
	group := app.Group(prefix, middlewares...)
	handler := NewConversationTagHandler(c)
	group.Get("/:id/tags", middleware.Tenant(handler.List))
	group.Post("/:id/tags", middleware.Tenant(handler.Attach))
	group.Delete("/:id/tags/:tagId", middleware.Tenant(handler.Detach))
}
