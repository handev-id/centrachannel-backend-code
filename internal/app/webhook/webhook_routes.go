package webhook

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(app *fiber.App, handler *WebhookHandler) {
	app.Post("/webhook/evolution", handler.HandleEvolution)
	app.Get("/webhook/meta", handler.HandleMetaVerify)
	app.Post("/webhook/meta", handler.HandleMetaWebhook)
}
