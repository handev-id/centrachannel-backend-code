package campaign

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *CampaignHandler) {
	group.Get("/", handler.List)
	group.Post("/", handler.Store)
	group.Get("/:id", handler.Show)
	group.Put("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
	group.Post("/:id/send", handler.Send)

	group.Get("/templates", handler.ListTemplates)
	group.Post("/templates", handler.CreateTemplate)
	group.Get("/templates/:id", handler.GetTemplate)
	group.Put("/templates/:id", handler.UpdateTemplate)
	group.Delete("/templates/:id", handler.DeleteTemplate)

	group.Get("/recipient-lists", handler.ListRecipientLists)
	group.Post("/recipient-lists", handler.CreateRecipientList)
	group.Get("/recipient-lists/:id", handler.GetRecipientList)
	group.Put("/recipient-lists/:id", handler.UpdateRecipientList)
	group.Delete("/recipient-lists/:id", handler.DeleteRecipientList)
	group.Get("/recipient-lists/:id/contacts", handler.ListRecipientContacts)
	group.Post("/recipient-lists/:id/contacts", handler.AddRecipientContact)
	group.Delete("/recipient-lists/:id/contacts/:contactId", handler.RemoveRecipientContact)
}
