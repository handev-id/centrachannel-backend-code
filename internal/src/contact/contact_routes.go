package contact

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
)

func RegisterRoutes(app fiber.Router, prefix string, c *di.Container, middlewares ...any) {
	group := app.Group(prefix, middlewares...)
	handler := NewContactHandler(c)
	group.Get("/", middleware.Tenant(handler.List))
	group.Post("/", middleware.Tenant(handler.Store))
	group.Get("/export", middleware.Tenant(handler.ExportCSV))
	group.Post("/import", middleware.Tenant(handler.ImportCSV))
	group.Get("/:id", middleware.Tenant(handler.Show))
	group.Put("/:id", middleware.Tenant(handler.Update))
	group.Delete("/:id", middleware.Tenant(handler.Delete))
	group.Post("/:id/merge", middleware.Tenant(handler.Merge))
	group.Post("/:id/unmerge", middleware.Tenant(handler.Unmerge))
	group.Get("/:id/conversations", middleware.Tenant(handler.Conversations))
}
