package whatsapp_device

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(group fiber.Router, handler *WhatsAppDeviceHandler) {
	group.Get("/", middleware.Tenant(handler.List))
	group.Post("/", middleware.Tenant(handler.Store))
	group.Get("/:id", middleware.Tenant(handler.Show))
	group.Put("/:id", middleware.Tenant(handler.Update))
	group.Delete("/:id", middleware.Tenant(handler.Delete))
	group.Post("/:id/connect", middleware.Tenant(handler.Connect))
	group.Post("/:id/disconnect", middleware.Tenant(handler.Disconnect))
	group.Get("/:id/connection-state", middleware.Tenant(handler.ConnectionState))
	group.Post("/:id/scan", middleware.Tenant(handler.Scan))
}
