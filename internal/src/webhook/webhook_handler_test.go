package webhook

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/utils/logger"
)

func setupApp(apiKey, metaSecret string) *fiber.App {
	app := fiber.New()

	deviceRepo := &mockDeviceRepo{}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
			return nil, sql.ErrNoRows
		},
		createFunc: func(ctx context.Context, q contact.DBTX, c *contact.Contact) (int, error) {
			return 100, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
			return nil, sql.ErrNoRows
		},
		createFunc: func(ctx context.Context, q profile.DBTX, p *profile.Profile) (int, error) {
			return 200, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: 1, Type: channelType}, nil
		},
	}
	convRepo := &mockConvRepo{
		listFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*conversation.Conversation, int, error) {
			return nil, 0, nil
		},
		createFunc: func(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (int, error) {
			return 300, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID int) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) {
			return 400, nil
		},
	}
	tenantRepo := &mockMetaTenantRepo{}

	service := NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, tenantRepo, nil, logger.NewLogger("debug", "text"))
	handler := &WebhookHandler{service: service, apiKey: apiKey, metaSecret: metaSecret}

	app.Post("/webhook/evolution", handler.HandleEvolution)
	app.Get("/webhook/meta", handler.HandleMetaVerify)
	app.Post("/webhook/meta", handler.HandleMetaWebhook)

	return app
}

func TestHandleEvolution_InvalidAPIKey(t *testing.T) {
	app := setupApp("secret123", "")

	body := bytes.NewReader([]byte(`{"event":"connection.update"}`))
	req := httptest.NewRequest(http.MethodPost, "/webhook/evolution", body)
	req.Header.Set("apikey", "wrong-key")
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandleEvolution_MissingAPIKey(t *testing.T) {
	app := setupApp("secret123", "")

	body := bytes.NewReader([]byte(`{"event":"connection.update"}`))
	req := httptest.NewRequest(http.MethodPost, "/webhook/evolution", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandleMetaVerify_Success(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=subscribe&hub.verify_token=meta_secret_456&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "challenge_abc" {
		t.Errorf("expected 'challenge_abc', got %s", buf.String())
	}
}

func TestHandleMetaVerify_WrongToken(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestHandleMetaVerify_WrongMode(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=unsubscribe&hub.verify_token=meta_secret_456&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestHandleMetaWebhook_Success(t *testing.T) {
	app := setupApp("", "")

	payload := MetaWebhookPayload{
		Object: "page",
		Entry: []MetaWebhookEntry{{
			ID: "page_123",
			Messaging: []MetaWebhookMessage{{
				Sender:  &MetaSender{ID: "sender_456"},
				Message: &MetaMessage{MID: "mid_789", Text: "Hello"},
			}},
		}},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook/meta", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleMetaWebhook_Instagram(t *testing.T) {
	app := setupApp("", "")

	payload := MetaWebhookPayload{
		Object: "instagram",
		Entry: []MetaWebhookEntry{{
			ID: "ig_biz_123",
			Messaging: []MetaWebhookMessage{{
				Sender:  &MetaSender{ID: "ig_sender_456"},
				Message: &MetaMessage{MID: "ig_mid_789", Text: "Hello from IG"},
			}},
		}},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook/meta", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
