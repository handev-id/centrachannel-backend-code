package domain

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"centrachannel/internal/di"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

type DomainHandler struct{ service DomainService }

func NewDomainHandler(c *di.Container) *DomainHandler {
	repo := NewDomainRepository()
	service := NewDomainService(repo, c.DB, c.Config)
	return &DomainHandler{service: service}
}
func NewDomainHandlerWithService(service DomainService) *DomainHandler {
	return &DomainHandler{service: service}
}

func (h *DomainHandler) List(c fiber.Ctx, t *tenant.Tenant) error {
	domains, err := h.service.List(c.Context(), t.ID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", domains)
}
func (h *DomainHandler) Get(c fiber.Ctx, t *tenant.Tenant) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}
	d, err := h.service.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, "success", d)
}
func (h *DomainHandler) Store(c fiber.Ctx, t *tenant.Tenant) error {
	var req CreateDomainRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(&req); err != nil {
		return response.ValidationFailed(c, err)
	}
	d, err := h.service.Create(c.Context(), req, t.ID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Domain created", d)
}
func (h *DomainHandler) Update(c fiber.Ctx, t *tenant.Tenant) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}
	var req UpdateDomainRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(&req); err != nil {
		return response.ValidationFailed(c, err)
	}
	d, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Domain updated", d)
}
func (h *DomainHandler) Delete(c fiber.Ctx, t *tenant.Tenant) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}
	if err := h.service.Delete(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Domain deleted", nil)
}
