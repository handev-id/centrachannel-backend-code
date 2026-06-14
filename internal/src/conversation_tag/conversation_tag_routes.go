package conversation_tag

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *ConversationTagHandler) {
	group.Get("/:id/tags", handler.List)
	group.Post("/:id/tags", handler.Attach)
	group.Delete("/:id/tags/:tagId", handler.Detach)
}
