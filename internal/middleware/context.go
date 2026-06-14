package middleware

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

func Tenant(h func(fiber.Ctx, *tenant.Tenant) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		t, ok := c.Locals("tenant").(*tenant.Tenant)
		if !ok || t == nil {
			return response.InternalServerError(c, "tenant context not found")
		}
		return h(c, t)
	}
}

func GetUserID(c fiber.Ctx) int {
	uid, ok := c.Locals("user_id").(int)
	if !ok {
		response.InternalServerError(c, "user context not found")
		return 0
	}
	return uid
}
