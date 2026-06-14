package whatsapp_device

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *WhatsAppDeviceHandler) {
	group.Get("/", handler.List)
	group.Post("/", handler.Store)
	group.Get("/:id", handler.Show)
	group.Put("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
	group.Post("/:id/connect", handler.Connect)
	group.Post("/:id/disconnect", handler.Disconnect)
	group.Post("/:id/scan", handler.Scan)
}
