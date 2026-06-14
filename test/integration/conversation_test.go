package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/tenant"
)

type mockConversationService struct {
	listFunc     func(ctx context.Context, q conversation.ListConversationQuery, t *tenant.Tenant) (*conversation.PaginatedResponse, error)
	getByIDFunc  func(ctx context.Context, tenantID int, id int) (*conversation.Conversation, error)
	createFunc   func(ctx context.Context, req conversation.CreateConversationRequest, t *tenant.Tenant) (*conversation.Conversation, error)
	assignFunc   func(ctx context.Context, tenantID int, id int, agentID int) error
	unassignFunc func(ctx context.Context, tenantID int, id int) error
	resolveFunc  func(ctx context.Context, tenantID int, id int) error
	reopenFunc   func(ctx context.Context, tenantID int, id int) error
	markReadFunc func(ctx context.Context, tenantID int, id int) error
}

func (m *mockConversationService) List(ctx context.Context, q conversation.ListConversationQuery, t *tenant.Tenant) (*conversation.PaginatedResponse, error) {
	return m.listFunc(ctx, q, t)
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

type apiResponse struct {
	Meta struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"meta"`
	Data json.RawMessage `json:"data"`
}

func readBody(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("unmarshal error: %v\nbody: %s", err, string(body))
	}
}

func intPtr(i int) *int { return &i }

func TestConversationList_Success(t *testing.T) {
	mock := &mockConversationService{
		listFunc: func(_ context.Context, q conversation.ListConversationQuery, t *tenant.Tenant) (*conversation.PaginatedResponse, error) {
			return &conversation.PaginatedResponse{
				Meta: conversation.PaginationMeta{Total: 2, PerPage: 20, CurrentPage: 1, LastPage: 1, From: 1, To: 2},
				Data: []*conversation.Conversation{
					{ID: 1, TenantID: 1, Status: "unassigned", ProfileID: 1, ChannelID: 1},
					{ID: 2, TenantID: 1, Status: "assigned", ProfileID: 2, ChannelID: 1, AgentID: intPtr(1)},
				},
			}, nil
		},
	}

	handler := conversation.NewConversationHandlerWithService(mock)
	app := NewTestApp()
	app.Use(TestAuthMiddleware())
	conversation.RegisterRoutes(app.Group("/api/v1/tenant/conversations"), handler)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/conversations?page=1&limit=20", nil)
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
		Meta conversation.PaginationMeta `json:"meta"`
		Data json.RawMessage             `json:"data"`
	}
	if err := json.Unmarshal(env.Data, &paginated); err != nil {
		t.Fatal(err)
	}

	if paginated.Meta.Total != 2 {
		t.Errorf("expected total 2, got %d", paginated.Meta.Total)
	}
	if paginated.Meta.PerPage != 20 {
		t.Errorf("expected per_page 20, got %d", paginated.Meta.PerPage)
	}
	if paginated.Meta.CurrentPage != 1 {
		t.Errorf("expected current_page 1, got %d", paginated.Meta.CurrentPage)
	}

	var items []*conversation.Conversation
	if err := json.Unmarshal(paginated.Data, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(items))
	}
	if items[0].ID != 1 || items[0].Status != "unassigned" {
		t.Errorf("unexpected first item: %+v", items[0])
	}
	if items[1].ID != 2 || items[1].Status != "assigned" {
		t.Errorf("unexpected second item: %+v", items[1])
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

	var env apiResponse
	readBody(t, resp, &env)

	if env.Meta.Code != 201 {
		t.Errorf("expected meta.code 201, got %d", env.Meta.Code)
	}
	if env.Meta.Message != "Conversation created" {
		t.Errorf("expected meta.message 'Conversation created', got %q", env.Meta.Message)
	}

	var conv conversation.Conversation
	if err := json.Unmarshal(env.Data, &conv); err != nil {
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

	var env apiResponse
	readBody(t, resp, &env)

	if env.Meta.Code != 200 {
		t.Errorf("expected meta.code 200, got %d", env.Meta.Code)
	}

	var conv conversation.Conversation
	if err := json.Unmarshal(env.Data, &conv); err != nil {
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

	var env apiResponse
	readBody(t, resp, &env)

	if env.Meta.Code != 200 {
		t.Errorf("expected meta.code 200, got %d", env.Meta.Code)
	}
	if env.Meta.Message != "Conversation assigned" {
		t.Errorf("expected meta.message 'Conversation assigned', got %q", env.Meta.Message)
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

	var env apiResponse
	readBody(t, resp, &env)

	if env.Meta.Code != 200 {
		t.Errorf("expected meta.code 200, got %d", env.Meta.Code)
	}
	if env.Meta.Message != "Conversation resolved" {
		t.Errorf("expected meta.message 'Conversation resolved', got %q", env.Meta.Message)
	}
}
