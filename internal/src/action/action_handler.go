package action

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/response"
)

type ActionHandler struct {
	service *ActionService
}

func NewActionHandler(c *di.Container) *ActionHandler {
	convRepo := conversation.NewConversationRepository()
	msgRepo := message.NewMessageRepository()
	profRepo := profile.NewProfileRepository()
	deviceRepo := whatsapp_device.NewWhatsAppDeviceRepository()
	client := whatsapp_device.NewEvolutionClient(c.Config.EvolutionAPIURL, c.Config.EvolutionAPIKey, c.Logger)
	service := NewActionService(convRepo, msgRepo, profRepo, deviceRepo, client, c.DB, c.Logger)
	return &ActionHandler{service: service}
}

func (h *ActionHandler) MarkConversationRead(c fiber.Ctx, t *tenant.Tenant) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.MarkConversationRead(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.OK(c, "Conversation marked as read", nil)
}
