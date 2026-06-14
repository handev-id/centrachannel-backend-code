package dashboard

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *DashboardHandler) {
	group.Get("/stats", handler.Stats)
	group.Get("/chart", handler.Chart)
}
