package profile

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/utils/response"
)

type ProfileHandler struct {
	service ProfileService
}

func NewProfileHandler(c *di.Container) *ProfileHandler {
	repo := NewProfileRepository()
	service := NewProfileService(repo, c.DB)
	return &ProfileHandler{service: service}
}

func NewProfileHandlerWithService(service ProfileService) *ProfileHandler {
	return &ProfileHandler{service: service}
}

func (h *ProfileHandler) List(c fiber.Ctx) error {
	contactID, _ := strconv.Atoi(c.Query("contact_id"))
	channelID, _ := strconv.Atoi(c.Query("channel_id"))

	q := ListProfileQuery{
		ContactID: contactID,
		ChannelID: channelID,
	}

	profiles, err := h.service.List(c.Context(), q)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", profiles)
}

func (h *ProfileHandler) Show(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	profile, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, "success", profile)
}

func (h *ProfileHandler) Update(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateProfileRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	profile, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Profile updated", profile)
}
