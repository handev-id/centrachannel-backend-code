package tenant

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"centrachannel/internal/di"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/utils/response"
	"centrachannel/internal/utils/tenantutils"
)

type TenantHandler struct {
	service TenantService
	rdb     RedisClient
	logger  *logger.Logger
}

type RedisClient interface {
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

func NewTenantHandler(c *di.Container) *TenantHandler {
	repo := NewTenantRepository()
	service := NewTenantService(repo, c.DB, c.Config, c.Logger)
	h := &TenantHandler{service: service, logger: c.Logger}
	if c.Redis != nil {
		h.rdb = c.Redis
	}
	return h
}

func (h *TenantHandler) Get(c fiber.Ctx) error {
	t, ok := c.Locals("tenant").(*Tenant)
	if !ok || t == nil {
		return response.InternalServerError(c, "tenant context not found")
	}

	resp, err := tenantutils.DeepCopyJSON(t)
	if err != nil {
		return response.InternalServerError(c, "Failed to copy tenant data")
	}
	resp.Settings = tenantutils.StripSettings(resp.Settings)

	domains, err := h.service.GetDomains(c.Context(), t.ID)
	if err != nil {
		return response.InternalServerError(c, "Failed to load tenant domains")
	}
	return response.OK(c, "Tenant retrieved successfully", &TenantResponse{Tenant: resp, Domains: domains})
}

func (h *TenantHandler) Update(c fiber.Ctx) error {
	t, ok := c.Locals("tenant").(*Tenant)
	if !ok || t == nil {
		return response.InternalServerError(c, "tenant context not found")
	}

	var req UpdateTenantRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(&req); err != nil {
		return response.ValidationFailed(c, err)
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Logo != nil {
		t.Logo = req.Logo
	}
	if req.Address != nil {
		t.Address = req.Address
	}
	if req.Phone != nil {
		t.Phone = req.Phone
	}
	if req.Email != nil {
		t.Email = req.Email
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
	}
	if req.Settings != nil {
		t.Settings = req.Settings
	}

	if err := h.service.Update(c.Context(), t); err != nil {
		return response.InternalServerError(c, err.Error())
	}

	// Invalidate Redis cache so the next request loads fresh data
	if h.rdb != nil {
		if err := h.rdb.Del(c.Context(), "tenant:v2:"+t.ID.String()).Err(); err != nil {
			h.logger.Error("tenant cache invalidation failed: %v", err)
		}
	}

	resp, err := tenantutils.DeepCopyJSON(t)
	if err != nil {
		return response.InternalServerError(c, "Failed to copy tenant data")
	}
	resp.Settings = tenantutils.StripSettings(resp.Settings)

	domains, err := h.service.GetDomains(c.Context(), t.ID)
	if err != nil {
		return response.InternalServerError(c, "Failed to load tenant domains")
	}
	return response.OK(c, "Tenant updated successfully", &TenantResponse{Tenant: resp, Domains: domains})
}
