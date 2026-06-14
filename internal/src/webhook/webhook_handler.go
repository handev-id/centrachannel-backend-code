package webhook

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/config"
	"centrachannel/internal/di"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/response"
	"centrachannel/internal/ws"
)

type WebhookHandler struct {
	service  WebhookService
	apiKey   string
	metaSecret string
}

func NewWebhookHandler(c *di.Container, cfg *config.Config, notifier ...ws.Notifier) *WebhookHandler {
	deviceRepo := whatsapp_device.NewWhatsAppDeviceRepository()
	contactRepo := contact.NewContactRepository()
	profileRepo := profile.NewProfileRepository()
	channelRepo := channel.NewChannelRepository()
	convRepo := conversation.NewConversationRepository()
	msgRepo := message.NewMessageRepository()
	tenantRepo := tenant.NewTenantRepository()
	service := NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, tenantRepo, c.DB, c.Logger, notifier...)
	return &WebhookHandler{service: service, apiKey: cfg.EvolutionAPIKey, metaSecret: cfg.MetaWebhookSecret}
}

func (h *WebhookHandler) HandleEvolution(c fiber.Ctx) error {
	key := c.Get("apikey")
	if key == "" {
		key = c.Get("x-api-key")
	}
	if h.apiKey != "" && key != h.apiKey {
		return response.Unauthorized(c, "invalid api key")
	}

	var payload EvolutionWebhookPayload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequest(c, "invalid payload", nil)
	}

	if err := h.service.ProcessEvolutionEvent(c.Context(), &payload); err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "ok", nil)
}

func (h *WebhookHandler) HandleMetaVerify(c fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode != "subscribe" || token != h.metaSecret {
		return c.SendStatus(fiber.StatusForbidden)
	}

	return c.SendString(challenge)
}

func (h *WebhookHandler) HandleMetaWebhook(c fiber.Ctx) error {
	var payload MetaWebhookPayload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequest(c, "invalid payload", nil)
	}

	if err := h.service.ProcessMetaEvent(c.Context(), &payload); err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "ok", nil)
}
