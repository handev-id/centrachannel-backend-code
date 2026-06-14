package registration

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(router fiber.Router, handler *RegistrationHandler) {
	router.Post("/api/tenants/onboard", handler.Onboard)
}
