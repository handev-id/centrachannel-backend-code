package conversation

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(group fiber.Router, handler *ConversationHandler) {
	group.Get("/", middleware.Tenant(handler.List))
	group.Post("/", middleware.Tenant(handler.Store))
	group.Get("/:id", middleware.Tenant(handler.Show))
	group.Post("/:id/assign", middleware.Tenant(handler.Assign))
	group.Post("/:id/unassign", middleware.Tenant(handler.Unassign))
	group.Post("/:id/resolve", middleware.Tenant(handler.Resolve))
	group.Post("/:id/reopen", middleware.Tenant(handler.Reopen))
	group.Get("/unread", middleware.Tenant(handler.TotalUnread))
	group.Put("/:id/read", middleware.Tenant(handler.Read))
}
