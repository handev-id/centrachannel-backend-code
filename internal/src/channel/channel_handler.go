package channel

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/utils/response"
)

type ChannelHandler struct {
	service ChannelService
}

func NewChannelHandler(c *di.Container) *ChannelHandler {
	repo := NewChannelRepository()
	service := NewChannelService(repo, c.DB)
	return &ChannelHandler{service: service}
}

func (h *ChannelHandler) List(c fiber.Ctx) error {
	channels, err := h.service.List(c.Context())
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", channels)
}
