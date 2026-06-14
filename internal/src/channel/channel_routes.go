package channel

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *ChannelHandler) {
	group.Get("/", handler.List)
}
