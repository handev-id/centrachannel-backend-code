package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func LoggerMiddleware() fiber.Handler {
	return logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path}\n",
	})
}