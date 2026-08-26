package action

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(group fiber.Router, handler *ActionHandler) {
	group.Put("/conversations/:id/read", middleware.Tenant(handler.MarkConversationRead))
}
