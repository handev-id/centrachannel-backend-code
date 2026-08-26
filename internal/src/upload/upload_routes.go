package upload

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
)

func RegisterRoutes(app fiber.Router, prefix string, c *di.Container, middlewares ...any) {
	group := app.Group(prefix, middlewares...)
	handler := NewUploadHandler(c)
	group.Post("/", handler.Upload)
}
