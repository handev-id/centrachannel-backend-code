package campaign

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
)

type CampaignHandler struct {
	service CampaignService
}

func NewCampaignHandler(c *di.Container) *CampaignHandler {
	repo := NewCampaignRepository()
	service := NewCampaignService(repo, c.DB, c.Config, c.Logger)
	return &CampaignHandler{service: service}
}

func NewCampaignHandlerWithService(service CampaignService) *CampaignHandler {
	return &CampaignHandler{service: service}
}

func (h *CampaignHandler) List(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	q := ListCampaignQuery{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
	}

	result, err := h.service.List(c.Context(), q, t)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", result)
}

func (h *CampaignHandler) Show(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	campaign, err := h.service.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, "success", campaign)
}

func (h *CampaignHandler) Store(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	var req CreateCampaignRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	campaign, err := h.service.Create(c.Context(), req, t, userID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Campaign created", campaign)
}

func (h *CampaignHandler) Update(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateCampaignRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	campaign, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Campaign updated", campaign)
}

func (h *CampaignHandler) Delete(c fiber.Ctx) error {
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
	return response.OK(c, "Campaign deleted", nil)
}

func (h *CampaignHandler) Send(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Send(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Campaign send started", nil)
}
