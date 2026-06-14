package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"centrachannel/internal/src/message"
)

type mockMessageService struct {
	listFunc       func(ctx context.Context, conversationID int, q message.ListMessageQuery) (*message.PaginatedResponse, error)
	sendFunc       func(ctx context.Context, req message.SendMessageRequest, tenantID int, conversationID int) (*message.Message, error)
	updateStatusFunc func(ctx context.Context, id int, status string) error
}

func (m *mockMessageService) List(ctx context.Context, conversationID int, q message.ListMessageQuery) (*message.PaginatedResponse, error) {
	return m.listFunc(ctx, conversationID, q)
}
func (m *mockMessageService) Send(ctx context.Context, req message.SendMessageRequest, tenantID int, conversationID int) (*message.Message, error) {
	return m.sendFunc(ctx, req, tenantID, conversationID)
}
func (m *mockMessageService) UpdateStatus(ctx context.Context, id int, status string) error {
	return m.updateStatusFunc(ctx, id, status)
}

func TestMessageList_Success(t *testing.T) {
	text := "Hello, world!"
	mock := &mockMessageService{
		listFunc: func(_ context.Context, conversationID int, q message.ListMessageQuery) (*message.PaginatedResponse, error) {
			return &message.PaginatedResponse{
				Meta: message.PaginationMeta{Total: 1, PerPage: 50, CurrentPage: 1, LastPage: 1, From: 1, To: 1},
				Data: []*message.Message{
					{
						ID: 1, TenantID: 1, ConversationID: conversationID,
						Text: &text, Status: "sent", SenderID: 1, SenderType: "user",
					},
				},
			}, nil
		},
	}

	handler := message.NewMessageHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	message.RegisterConversationRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/conversations/1/messages?page=1&limit=50", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var env apiResponse
	readBody(t, resp, &env)

	if env.Meta.Code != 200 {
		t.Errorf("expected meta.code 200, got %d", env.Meta.Code)
	}
	if env.Meta.Message != "success" {
		t.Errorf("expected meta.message 'success', got %q", env.Meta.Message)
	}

	var paginated struct {
		Meta message.PaginationMeta `json:"meta"`
		Data json.RawMessage        `json:"data"`
	}
	if err := json.Unmarshal(env.Data, &paginated); err != nil {
		t.Fatal(err)
	}

	if paginated.Meta.Total != 1 {
		t.Errorf("expected total 1, got %d", paginated.Meta.Total)
	}
	if paginated.Meta.PerPage != 50 {
		t.Errorf("expected per_page 50, got %d", paginated.Meta.PerPage)
	}
	if paginated.Meta.CurrentPage != 1 {
		t.Errorf("expected current_page 1, got %d", paginated.Meta.CurrentPage)
	}

	var items []*message.Message
	if err := json.Unmarshal(paginated.Data, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(items))
	}
	if items[0].ID != 1 {
		t.Errorf("expected message id 1, got %d", items[0].ID)
	}
	if items[0].ConversationID != 1 {
		t.Errorf("expected conversation_id 1, got %d", items[0].ConversationID)
	}
	if items[0].SenderType != "user" {
		t.Errorf("expected sender_type 'user', got %q", items[0].SenderType)
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

	var env apiResponse
	readBody(t, resp, &env)

	if env.Meta.Code != 201 {
		t.Errorf("expected meta.code 201, got %d", env.Meta.Code)
	}
	if env.Meta.Message != "Message sent" {
		t.Errorf("expected meta.message 'Message sent', got %q", env.Meta.Message)
	}

	var msg message.Message
	if err := json.Unmarshal(env.Data, &msg); err != nil {
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
