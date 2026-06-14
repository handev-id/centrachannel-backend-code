package middleware

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/utils/response"
)

func RequireRole(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRoles, ok := c.Locals("roles").([]string)
		if !ok || len(userRoles) == 0 {
			return response.Forbidden(c, "Insufficient permissions")
		}

		roleSet := make(map[string]bool, len(userRoles))
		for _, r := range userRoles {
			roleSet[r] = true
		}

		for _, required := range roles {
			if roleSet[required] {
				return c.Next()
			}
		}
		return response.Forbidden(c, "Insufficient permissions")
	}
}

func IsAgentOnly(c fiber.Ctx) bool {
	roles, ok := c.Locals("roles").([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == "super-admin" || r == "admin" {
			return false
		}
	}
	return true
}
