package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"centrachannel/internal/utils/response"
)

type Tenant struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	Domain    string          `json:"domain"`
	Logo      json.RawMessage `json:"logo,omitempty"`
	Address   *string         `json:"address,omitempty"`
	Phone     *string         `json:"phone,omitempty"`
	Email     *string         `json:"email,omitempty"`
	IsActive  bool            `json:"is_active"`
	Settings  json.RawMessage `json:"settings,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func TenantMiddleware(rdb *redis.Client, db *sql.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		domain := extractDomain(c.Hostname())
		if domain == "" {
			domain = c.Get("X-Tenant-Domain")
		}
		if domain == "" {
			return response.BadRequest(c, "Tenant domain is required", nil)
		}

		tenant, err := getTenant(c.Context(), rdb, db, domain)
		if err != nil {
			return response.NotFound(c, "Tenant not found")
		}

		c.Locals("tenant", tenant)
		return c.Next()
	}
}

func extractDomain(host string) string {
	host = strings.Split(host, ":")[0]
	return host
}

func getTenant(ctx context.Context, rdb *redis.Client, db *sql.DB, domain string) (*Tenant, error) {
	cacheKey := fmt.Sprintf("tenant:%s", domain)

	data, err := rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var t Tenant
		if json.Unmarshal(data, &t) == nil {
			return &t, nil
		}
	}

	var t Tenant
	var logo, settings sql.NullString
	var address, phone, email sql.NullString

	err = db.QueryRowContext(ctx,
		`SELECT id, name, domain, logo, address, phone, email, is_active, settings, created_at, updated_at FROM tenants WHERE domain = $1 AND is_active = TRUE`,
		domain,
	).Scan(&t.ID, &t.Name, &t.Domain, &logo, &address, &phone, &email, &t.IsActive, &settings, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if logo.Valid {
		t.Logo = json.RawMessage(logo.String)
	}
	if address.Valid {
		t.Address = &address.String
	}
	if phone.Valid {
		t.Phone = &phone.String
	}
	if email.Valid {
		t.Email = &email.String
	}
	if settings.Valid {
		t.Settings = json.RawMessage(settings.String)
	}

	if cacheData, err := json.Marshal(t); err == nil {
		rdb.Set(ctx, cacheKey, cacheData, time.Hour)
	}

	return &t, nil
}
