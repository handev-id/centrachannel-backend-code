package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type mockWebhookService struct {
	processEvolutionFunc func(ctx context.Context, payload *EvolutionWebhookPayload) error
	processMetaFunc      func(ctx context.Context, payload *MetaWebhookPayload) error
}

func (m *mockWebhookService) ProcessEvolutionEvent(ctx context.Context, payload *EvolutionWebhookPayload) error {
	if m.processEvolutionFunc != nil {
		return m.processEvolutionFunc(ctx, payload)
	}
	return nil
}

func (m *mockWebhookService) ProcessMetaEvent(ctx context.Context, payload *MetaWebhookPayload) error {
	if m.processMetaFunc != nil {
		return m.processMetaFunc(ctx, payload)
	}
	return nil
}

func setupApp(apiKey, metaSecret string) *fiber.App {
	app := fiber.New()
	handler := &WebhookHandler{service: &mockWebhookService{}, apiKey: apiKey, metaSecret: metaSecret}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandleMetaVerify_Success(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=subscribe&hub.verify_token=meta_secret_456&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestHandleMetaVerify_WrongMode(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=unsubscribe&hub.verify_token=meta_secret_456&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
