package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

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
	service    WebhookService
	apiKey     string
	metaSecret string
	metaAppSecret string
}

func NewWebhookHandler(c *di.Container, cfg *config.Config, notifier ws.Notifier) *WebhookHandler {
	deviceRepo := whatsapp_device.NewWhatsAppDeviceRepository()
	contactRepo := contact.NewContactRepository()
	profileRepo := profile.NewProfileRepository()
	channelRepo := channel.NewChannelRepository()
	convRepo := conversation.NewConversationRepository()
	msgRepo := message.NewMessageRepository()
	tenantRepo := tenant.NewTenantRepository()
	service := NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, tenantRepo, c.DB, c.Logger, c.Redis, notifier)
	return &WebhookHandler{service: service, apiKey: cfg.EvolutionAPIKey, metaSecret: cfg.MetaWebhookSecret, metaAppSecret: cfg.MetaAppSecret}
}

func (h *WebhookHandler) HandleEvolution(c fiber.Ctx) error {
	key := c.Get("apikey")
	if key == "" {
		key = c.Get("x-api-key")
	}
	if h.apiKey == "" || key == "" || key != h.apiKey {
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
	if h.metaAppSecret == "" {
		return response.Unauthorized(c, "meta webhook secret is not configured")
	}
	signature := c.Get("x-hub-signature-256")
	const prefix = "sha256="
	if len(signature) <= len(prefix) || signature[:len(prefix)] != prefix {
		return response.Unauthorized(c, "invalid meta webhook signature")
	}
	mac := hmac.New(sha256.New, []byte(h.metaAppSecret))
	_, _ = mac.Write(c.Body())
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature[len(prefix):]), []byte(expected)) {
		return response.Unauthorized(c, "invalid meta webhook signature")
	}

	var payload MetaWebhookPayload
	if err := c.Bind().Body(&payload); err != nil {
		return response.BadRequest(c, "invalid payload", nil)
	}

	if err := h.service.ProcessMetaEvent(c.Context(), &payload); err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "ok", nil)
}
