package campaign

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
)

func RegisterRoutes(group fiber.Router, handler *CampaignHandler) {
	group.Get("/", middleware.Tenant(handler.List))
	group.Post("/", middleware.Tenant(handler.Store))
	group.Get("/:id", middleware.Tenant(handler.Show))
	group.Put("/:id", middleware.Tenant(handler.Update))
	group.Delete("/:id", middleware.Tenant(handler.Delete))
	group.Post("/:id/send", middleware.Tenant(handler.Send))

	group.Get("/templates", middleware.Tenant(handler.ListTemplates))
	group.Post("/templates", middleware.Tenant(handler.CreateTemplate))
	group.Get("/templates/:id", middleware.Tenant(handler.GetTemplate))
	group.Put("/templates/:id", middleware.Tenant(handler.UpdateTemplate))
	group.Delete("/templates/:id", middleware.Tenant(handler.DeleteTemplate))

	group.Get("/recipient-lists", middleware.Tenant(handler.ListRecipientLists))
	group.Post("/recipient-lists", middleware.Tenant(handler.CreateRecipientList))
	group.Get("/recipient-lists/:id", middleware.Tenant(handler.GetRecipientList))
	group.Put("/recipient-lists/:id", middleware.Tenant(handler.UpdateRecipientList))
	group.Delete("/recipient-lists/:id", middleware.Tenant(handler.DeleteRecipientList))
	group.Get("/recipient-lists/:id/contacts", handler.ListRecipientContacts)
	group.Post("/recipient-lists/:id/contacts", handler.AddRecipientContact)
	group.Delete("/recipient-lists/:id/contacts/:contactId", handler.RemoveRecipientContact)
}
