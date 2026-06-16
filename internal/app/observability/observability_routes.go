package observability

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(app *fiber.App, handler *ObservabilityHandler) {
	app.Get("/health", handler.Health)
	app.Get("/ping", handler.Ping)
	app.Get("/version", handler.Version)
	app.Get("/sse-test-ping", handler.SSETestPing)
}
