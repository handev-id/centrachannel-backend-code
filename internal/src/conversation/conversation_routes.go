package conversation

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *ConversationHandler) {
	group.Get("/", handler.List)
	group.Post("/", handler.Store)
	group.Get("/:id", handler.Show)
	group.Post("/:id/assign", handler.Assign)
	group.Post("/:id/unassign", handler.Unassign)
	group.Post("/:id/resolve", handler.Resolve)
	group.Post("/:id/reopen", handler.Reopen)
	group.Put("/:id/read", handler.Read)
}
