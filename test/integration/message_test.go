package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"centrachannel/internal/src/message"
)

type mockMessageService struct {
	listCursorFunc   func(ctx context.Context, conversationID int, q message.ListMessageQuery) ([]*message.Message, int, bool, error)
	sendFunc         func(ctx context.Context, req message.SendMessageRequest, tenantID int, conversationID int) (*message.Message, error)
	updateStatusFunc func(ctx context.Context, id int, status string) error
}

func (m *mockMessageService) ListCursor(ctx context.Context, conversationID int, q message.ListMessageQuery) ([]*message.Message, int, bool, error) {
	return m.listCursorFunc(ctx, conversationID, q)
}
func (m *mockMessageService) Send(ctx context.Context, req message.SendMessageRequest, tenantID int, conversationID int) (*message.Message, error) {
	return m.sendFunc(ctx, req, tenantID, conversationID)
}
func (m *mockMessageService) UpdateStatus(ctx context.Context, id int, status string) error {
	return m.updateStatusFunc(ctx, id, status)
}

func TestMessageListCursor_Success(t *testing.T) {
	text := "Older message"
	mock := &mockMessageService{
		listCursorFunc: func(_ context.Context, conversationID int, q message.ListMessageQuery) ([]*message.Message, int, bool, error) {
			return []*message.Message{
				{ID: 40, TenantID: 1, ConversationID: conversationID, Text: &text, Status: "sent", SenderID: 1, SenderType: "user"},
			}, 40, true, nil
		},
	}

	handler := message.NewMessageHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	message.RegisterConversationRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/conversations/1/messages?last_id=50&limit=50", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		Meta struct {
			LastID  int  `json:"last_id"`
			HasMore bool `json:"has_more"`
		} `json:"meta"`
		Data json.RawMessage `json:"data"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal error: %v\nbody: %s", err, string(body))
	}

	if result.Meta.LastID != 40 {
		t.Errorf("expected last_id 40, got %d", result.Meta.LastID)
	}
	if !result.Meta.HasMore {
		t.Error("expected has_more true")
	}

	var items []*message.Message
	if err := json.Unmarshal(result.Data, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(items))
	}
	if items[0].ID != 40 {
		t.Errorf("expected message id 40, got %d", items[0].ID)
	}
}

func TestMessageSend_Success(t *testing.T) {
	text := "Hello from test"
	mock := &mockMessageService{
		sendFunc: func(_ context.Context, req message.SendMessageRequest, tenantID int, conversationID int) (*message.Message, error) {
			return &message.Message{
				ID: 1, TenantID: tenantID, ConversationID: conversationID,
				Text: &text, Status: "sent", SenderID: req.SenderID, SenderType: req.SenderType,
			}, nil
		},
	}

	handler := message.NewMessageHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	message.RegisterConversationRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	body := message.SendMessageRequest{
		Text: &text, SenderID: 1, SenderType: "user",
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/conversations/1/messages", JSONBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var msg message.Message
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		t.Fatal(err)
	}

	if msg.ID != 1 {
		t.Errorf("expected message id 1, got %d", msg.ID)
	}
	if msg.TenantID != 1 {
		t.Errorf("expected tenant_id 1, got %d", msg.TenantID)
	}
	if msg.ConversationID != 1 {
		t.Errorf("expected conversation_id 1, got %d", msg.ConversationID)
	}
	if msg.Status != "sent" {
		t.Errorf("expected status 'sent', got %q", msg.Status)
	}
	if msg.SenderID != 1 {
		t.Errorf("expected sender_id 1, got %d", msg.SenderID)
	}
	if msg.SenderType != "user" {
		t.Errorf("expected sender_type 'user', got %q", msg.SenderType)
	}
}
