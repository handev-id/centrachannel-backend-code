package dashboard

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
)

func RegisterRoutes(app fiber.Router, prefix string, c *di.Container, middlewares ...any) {
	group := app.Group(prefix, middlewares...)
	handler := NewDashboardHandler(c)
	group.Get("/", middleware.Tenant(handler.Stats))
	group.Get("/chart", middleware.Tenant(handler.Chart))
}
