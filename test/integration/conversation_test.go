package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/tenant"
)

type mockConversationService struct {
	listCursorFunc       func(ctx context.Context, q conversation.ListConversationQuery, t *tenant.Tenant) ([]*conversation.Conversation, int, string, bool, error)
	getByIDFunc          func(ctx context.Context, tenantID int, id int) (*conversation.Conversation, error)
	createFunc           func(ctx context.Context, req conversation.CreateConversationRequest, t *tenant.Tenant) (*conversation.Conversation, error)
	assignFunc           func(ctx context.Context, tenantID int, id int, agentID int) error
	unassignFunc         func(ctx context.Context, tenantID int, id int) error
	resolveFunc          func(ctx context.Context, tenantID int, id int) error
	reopenFunc           func(ctx context.Context, tenantID int, id int) error
	markReadFunc         func(ctx context.Context, tenantID int, id int) error
	getTotalUnreadFunc   func(ctx context.Context, tenantID int) (int, error)
}

func (m *mockConversationService) ListCursor(ctx context.Context, q conversation.ListConversationQuery, t *tenant.Tenant) ([]*conversation.Conversation, int, string, bool, error) {
	return m.listCursorFunc(ctx, q, t)
}
func (m *mockConversationService) GetByID(ctx context.Context, tenantID int, id int) (*conversation.Conversation, error) {
	return m.getByIDFunc(ctx, tenantID, id)
}
func (m *mockConversationService) Create(ctx context.Context, req conversation.CreateConversationRequest, t *tenant.Tenant) (*conversation.Conversation, error) {
	return m.createFunc(ctx, req, t)
}
func (m *mockConversationService) Assign(ctx context.Context, tenantID int, id int, agentID int) error {
	return m.assignFunc(ctx, tenantID, id, agentID)
}
func (m *mockConversationService) Unassign(ctx context.Context, tenantID int, id int) error {
	return m.unassignFunc(ctx, tenantID, id)
}
func (m *mockConversationService) Resolve(ctx context.Context, tenantID int, id int) error {
	return m.resolveFunc(ctx, tenantID, id)
}
func (m *mockConversationService) Reopen(ctx context.Context, tenantID int, id int) error {
	return m.reopenFunc(ctx, tenantID, id)
}
func (m *mockConversationService) MarkRead(ctx context.Context, tenantID int, id int) error {
	return m.markReadFunc(ctx, tenantID, id)
}
func (m *mockConversationService) GetTotalUnread(ctx context.Context, tenantID int) (int, error) {
	return m.getTotalUnreadFunc(ctx, tenantID)
}

func TestConversationListCursor_Success(t *testing.T) {
	activity := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	lastName := "Santoso"
	mock := &mockConversationService{
		listCursorFunc: func(_ context.Context, q conversation.ListConversationQuery, t *tenant.Tenant) ([]*conversation.Conversation, int, string, bool, error) {
			return []*conversation.Conversation{
				{ID: 12, TenantID: 1, Status: "assigned", ProfileID: 1, ChannelID: 1, LastActivity: &activity,
					Contact: &conversation.ConversationContact{ID: 7, FirstName: "Budi", LastName: &lastName, Avatar: json.RawMessage(`{"url":"https://cdn.example.com/avatar.png"}`)}},
				{ID: 11, TenantID: 1, Status: "unassigned", ProfileID: 2, ChannelID: 1, LastActivity: &activity},
			}, 11, activity.Format(time.RFC3339), true, nil
		},
	}

	handler := conversation.NewConversationHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	conversation.RegisterRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/conversations?last_id=13&last_activity=2026-08-02T11:00:00Z", nil)
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
			LastID       int    `json:"last_id"`
			LastActivity string `json:"last_activity"`
			HasMore      bool   `json:"has_more"`
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

	if result.Meta.LastID != 11 {
		t.Errorf("expected last_id 11, got %d", result.Meta.LastID)
	}
	if !result.Meta.HasMore {
		t.Error("expected has_more true")
	}
	if result.Meta.LastActivity == "" {
		t.Error("expected last_activity to be present")
	}

	var items []*conversation.Conversation
	if err := json.Unmarshal(result.Data, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(items))
	}
	if items[0].ID != 12 {
		t.Errorf("expected first item id 12, got %d", items[0].ID)
	}
	if items[0].Contact == nil {
		t.Fatal("expected contact on first item")
	}
	if items[0].Contact.ID != 7 || items[0].Contact.FirstName != "Budi" || items[0].Contact.LastName == nil || *items[0].Contact.LastName != "Santoso" {
		t.Errorf("unexpected contact: %+v", items[0].Contact)
	}
	if len(items[0].Contact.Avatar) == 0 {
		t.Error("expected contact avatar to be present")
	}
	if items[1].Contact != nil {
		t.Errorf("expected no contact on second item, got %+v", items[1].Contact)
	}
}

func TestConversationListCursor_Validation(t *testing.T) {
	mock := &mockConversationService{
		listCursorFunc: func(_ context.Context, _ conversation.ListConversationQuery, _ *tenant.Tenant) ([]*conversation.Conversation, int, string, bool, error) {
			return []*conversation.Conversation{}, 0, "", false, nil
		},
	}

	handler := conversation.NewConversationHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	conversation.RegisterRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	tests := []struct {
		name string
		url  string
		want int
	}{
		{"last_id without last_activity", "/api/v1/tenant/conversations?last_id=10", http.StatusBadRequest},
		{"last_activity without last_id", "/api/v1/tenant/conversations?last_activity=2026-08-02T10%3A00%3A00Z", http.StatusBadRequest},
		{"both present", "/api/v1/tenant/conversations?last_id=10&last_activity=2026-08-02T10%3A00%3A00Z", http.StatusOK},
		{"neither present", "/api/v1/tenant/conversations", http.StatusOK},
		{"empty last_activity with last_id (null tail)", "/api/v1/tenant/conversations?last_id=10&last_activity=", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.want {
				t.Errorf("expected status %d, got %d", tt.want, resp.StatusCode)
			}
		})
	}
}

func TestConversationCreate_Success(t *testing.T) {
	mock := &mockConversationService{
		createFunc: func(_ context.Context, req conversation.CreateConversationRequest, t *tenant.Tenant) (*conversation.Conversation, error) {
			return &conversation.Conversation{
				ID: 1, TenantID: t.ID, Status: "unassigned",
				ProfileID: req.ProfileID, ChannelID: req.ChannelID,
			}, nil
		},
	}

	handler := conversation.NewConversationHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	conversation.RegisterRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	body := conversation.CreateConversationRequest{ProfileID: 10, ChannelID: 5}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/conversations", JSONBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var conv conversation.Conversation
	if err := json.NewDecoder(resp.Body).Decode(&conv); err != nil {
		t.Fatal(err)
	}

	if conv.ID != 1 {
		t.Errorf("expected conversation id 1, got %d", conv.ID)
	}
	if conv.TenantID != 1 {
		t.Errorf("expected tenant_id 1, got %d", conv.TenantID)
	}
	if conv.Status != "unassigned" {
		t.Errorf("expected status 'unassigned', got %q", conv.Status)
	}
	if conv.ProfileID != 10 {
		t.Errorf("expected profile_id 10, got %d", conv.ProfileID)
	}
	if conv.ChannelID != 5 {
		t.Errorf("expected channel_id 5, got %d", conv.ChannelID)
	}
}

func TestConversationShow_Success(t *testing.T) {
	mock := &mockConversationService{
		getByIDFunc: func(_ context.Context, tenantID int, id int) (*conversation.Conversation, error) {
			return &conversation.Conversation{
				ID: id, TenantID: tenantID, Status: "unassigned",
				ProfileID: 10, ChannelID: 5,
			}, nil
		},
	}

	handler := conversation.NewConversationHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	conversation.RegisterRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/conversations/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var conv conversation.Conversation
	if err := json.NewDecoder(resp.Body).Decode(&conv); err != nil {
		t.Fatal(err)
	}

	if conv.ID != 1 {
		t.Errorf("expected conversation id 1, got %d", conv.ID)
	}
	if conv.TenantID != 1 {
		t.Errorf("expected tenant_id 1, got %d", conv.TenantID)
	}
	if conv.ProfileID != 10 {
		t.Errorf("expected profile_id 10, got %d", conv.ProfileID)
	}
}

func TestConversationAssign_Success(t *testing.T) {
	mock := &mockConversationService{
		assignFunc: func(_ context.Context, tenantID int, id int, agentID int) error {
			return nil
		},
	}

	handler := conversation.NewConversationHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	conversation.RegisterRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/conversations/1/assign", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestConversationResolve_Success(t *testing.T) {
	mock := &mockConversationService{
		resolveFunc: func(_ context.Context, tenantID int, id int) error {
			return nil
		},
	}

	handler := conversation.NewConversationHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	conversation.RegisterRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/conversations/1/resolve", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
