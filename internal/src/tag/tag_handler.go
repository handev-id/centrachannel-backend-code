package tag

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
)

type TagHandler struct {
	service TagService
}

func NewTagHandler(c *di.Container) *TagHandler {
	repo := NewTagRepository()
	service := NewTagService(repo, c.DB, c.Config, c.Logger)
	return &TagHandler{service: service}
}

func (h *TagHandler) List(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	tags, err := h.service.List(c.Context(), t.ID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", tags)
}

func (h *TagHandler) Store(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	var req CreateTagRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	tag, err := h.service.Create(c.Context(), req, t.ID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Tag created", tag)
}

func (h *TagHandler) Update(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateTagRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	tag, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Tag updated", tag)
}

func (h *TagHandler) Delete(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Delete(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Tag deleted", nil)
}
