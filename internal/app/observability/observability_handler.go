package observability

import (
	"database/sql"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"centrachannel/config"
	"centrachannel/internal/utils/response"
)

type ObservabilityHandler struct {
	config *config.Config
	db     *sql.DB
}

func NewObservabilityHandler(cfg *config.Config, db *sql.DB) *ObservabilityHandler {
	return &ObservabilityHandler{config: cfg, db: db}
}

func (h *ObservabilityHandler) Health(c fiber.Ctx) error {
	return response.OK(c, "healthy", fiber.Map{"status": "ok"})
}

func (h *ObservabilityHandler) Ping(c fiber.Ctx) error {
	return c.SendString("pong")
}

func (h *ObservabilityHandler) Version(c fiber.Ctx) error {
	return response.OK(c, "ok", fiber.Map{"version": "1.0.0", "app": "CentraChannel API"})
}

func (h *ObservabilityHandler) AskTLS(c fiber.Ctx) error {
	domain := strings.ToLower(strings.TrimSpace(c.Query("domain")))
	if domain == "" {
		return c.SendStatus(fiber.StatusForbidden)
	}

	parsed, err := url.Parse("https://" + domain)
	if err != nil || parsed.Hostname() != domain {
		return c.SendStatus(fiber.StatusForbidden)
	}

	baseDomain := strings.ToLower(strings.TrimSpace(h.config.BaseDomain))
	if domain == baseDomain {
		return c.SendStatus(fiber.StatusOK)
	}

	if strings.HasSuffix(domain, "."+baseDomain) {
		var exists int
		err := h.db.QueryRowContext(c.Context(),
			`SELECT 1 FROM tenants WHERE domain = $1 AND is_active = TRUE`,
			domain,
		).Scan(&exists)
		if err == nil && exists == 1 {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.SendStatus(fiber.StatusForbidden)
	}

	for _, allowedDomain := range h.config.TLSAllowedDomains {
		if domain == allowedDomain {
			return c.SendStatus(fiber.StatusOK)
		}
	}

	return c.SendStatus(fiber.StatusForbidden)
}
