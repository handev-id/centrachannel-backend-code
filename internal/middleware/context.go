package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/src/tenant"
)

func GetTenant(c fiber.Ctx) (*tenant.Tenant, error) {
	t, ok := c.Locals("tenant").(*tenant.Tenant)
	if !ok || t == nil {
		return nil, fmt.Errorf("tenant context not found")
	}
	return t, nil
}

func GetUserID(c fiber.Ctx) (int, error) {
	uid, ok := c.Locals("user_id").(int)
	if !ok {
		return 0, fmt.Errorf("user context not found")
	}
	return uid, nil
}
