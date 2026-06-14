package conversation_tag

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(group fiber.Router, handler *ConversationTagHandler) {
	group.Get("/:id/tags", middleware.Tenant(handler.List))
	group.Post("/:id/tags", middleware.Tenant(handler.Attach))
	group.Delete("/:id/tags/:tagId", middleware.Tenant(handler.Detach))
}
