package middleware

import (
	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
	"database/sql"
	"errors"
	"github.com/gofiber/fiber/v3"
	"net"
	"strings"
)

func TenantMiddleware(resolver tenant.TenantResolver, cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		host := c.Hostname()
		if parsed, _, err := net.SplitHostPort(host); err == nil {
			host = parsed
		}
		// Only development requests may use the explicit tenant hostname header.
		if cfg.Env == "development" || cfg.Env == "local" {
			if override := c.Get("X-Tenant-Domain"); override != "" {
				host = override
			}
		}
		host = strings.TrimSpace(host)
		if host == "" {
			return response.BadRequest(c, "Invalid tenant domain", nil)
		}
		t, err := resolver.Resolve(c.Context(), host)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return response.NotFound(c, "Tenant not found")
			}
			return response.InternalServerError(c, "Failed to resolve tenant")
		}
		c.Locals("tenant", t)
		return c.Next()
	}
}
