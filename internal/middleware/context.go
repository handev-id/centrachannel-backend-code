package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

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

func GetUserID(c fiber.Ctx) (uuid.UUID, error) {
	uid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok || uid == uuid.Nil {
		return uuid.Nil, fmt.Errorf("user context not found")
	}
	return uid, nil
}
