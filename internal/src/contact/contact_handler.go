package contact

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
)

type ContactHandler struct {
	service ContactService
}

func NewContactHandler(c *di.Container) *ContactHandler {
	repo := NewContactRepository()
	service := NewContactService(repo, c.DB, c.Config, c.Logger)
	return &ContactHandler{service: service}
}

func NewContactHandlerWithService(service ContactService) *ContactHandler {
	return &ContactHandler{service: service}
}

func (h *ContactHandler) List(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	channelID, _ := strconv.Atoi(c.Query("channel_id"))

	q := ListContactQuery{
		Page:      page,
		Limit:     limit,
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		ChannelID: channelID,
		SortBy:    c.Query("sort_by"),
	}

	result, err := h.service.List(c.Context(), q, t)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", result)
}

func (h *ContactHandler) Show(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	contact, err := h.service.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, "success", contact)
}

func (h *ContactHandler) Store(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	var req CreateContactRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	contact, err := h.service.Create(c.Context(), req, t)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Contact created", contact)
}

func (h *ContactHandler) Update(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateContactRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	contact, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Contact updated", contact)
}

func (h *ContactHandler) Delete(c fiber.Ctx) error {
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
	return response.OK(c, "Contact deleted", nil)
}

func (h *ContactHandler) Merge(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	sourceID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req MergeContactRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	if err := h.service.Merge(c.Context(), t.ID, sourceID, req.TargetContactID); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Contacts merged", nil)
}

func (h *ContactHandler) Unmerge(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Unmerge(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Contact unmerged", nil)
}

func (h *ContactHandler) Conversations(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	convs, err := h.service.GetConversations(c.Context(), t.ID, id)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", convs)
}
