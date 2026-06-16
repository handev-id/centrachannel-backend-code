package dashboard

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(group fiber.Router, handler *DashboardHandler) {
	group.Get("/", middleware.Tenant(handler.Stats))
	group.Get("/chart", middleware.Tenant(handler.Chart))
}
