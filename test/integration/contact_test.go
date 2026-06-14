package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/tenant"
)

type mockContactService struct {
	listFunc             func(ctx context.Context, q contact.ListContactQuery, t *tenant.Tenant) (*contact.PaginatedResponse, error)
	getByIDFunc          func(ctx context.Context, tenantID int, id int) (*contact.Contact, error)
	createFunc           func(ctx context.Context, req contact.CreateContactRequest, t *tenant.Tenant) (*contact.Contact, error)
	updateFunc           func(ctx context.Context, tenantID int, id int, req contact.UpdateContactRequest) (*contact.Contact, error)
	deleteFunc           func(ctx context.Context, tenantID int, id int) error
	getConversationsFunc func(ctx context.Context, tenantID int, contactID int) (interface{}, error)
	mergeFunc            func(ctx context.Context, tenantID int, sourceID int, targetID int) error
	unmergeFunc          func(ctx context.Context, tenantID int, id int) error
	importCSVFunc        func(ctx context.Context, tenantID int, records [][]string) (*contact.CSVImportResult, error)
	exportCSVFunc        func(ctx context.Context, tenantID int) (string, error)
}

func (m *mockContactService) List(ctx context.Context, q contact.ListContactQuery, t *tenant.Tenant) (*contact.PaginatedResponse, error) {
	return m.listFunc(ctx, q, t)
}

func (m *mockContactService) GetByID(ctx context.Context, tenantID int, id int) (*contact.Contact, error) {
	return m.getByIDFunc(ctx, tenantID, id)
}

func (m *mockContactService) Create(ctx context.Context, req contact.CreateContactRequest, t *tenant.Tenant) (*contact.Contact, error) {
	return m.createFunc(ctx, req, t)
}

func (m *mockContactService) Update(ctx context.Context, tenantID int, id int, req contact.UpdateContactRequest) (*contact.Contact, error) {
	return m.updateFunc(ctx, tenantID, id, req)
}

func (m *mockContactService) Delete(ctx context.Context, tenantID int, id int) error {
	return m.deleteFunc(ctx, tenantID, id)
}

func (m *mockContactService) GetConversations(ctx context.Context, tenantID int, contactID int) (interface{}, error) {
	return m.getConversationsFunc(ctx, tenantID, contactID)
}

func (m *mockContactService) Merge(ctx context.Context, tenantID int, sourceID int, targetID int) error {
	if m.mergeFunc != nil {
		return m.mergeFunc(ctx, tenantID, sourceID, targetID)
	}
	return nil
}

func (m *mockContactService) Unmerge(ctx context.Context, tenantID int, id int) error {
	if m.unmergeFunc != nil {
		return m.unmergeFunc(ctx, tenantID, id)
	}
	return nil
}
func (m *mockContactService) ExportCSV(ctx context.Context, tenantID int) (string, error) {
	if m.exportCSVFunc != nil {
		return m.exportCSVFunc(ctx, tenantID)
	}
	return "", nil
}

func (m *mockContactService) ImportCSV(ctx context.Context, tenantID int, records [][]string) (*contact.CSVImportResult, error) {
	if m.importCSVFunc != nil {
		return m.importCSVFunc(ctx, tenantID, records)
	}
	return &contact.CSVImportResult{}, nil
}

func strPtr(s string) *string { return &s }

func parseResp(t *testing.T, resp *http.Response, v interface{}) {
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

func TestContactList_200(t *testing.T) {
	mockSvc := &mockContactService{
		listFunc: func(ctx context.Context, q contact.ListContactQuery, t *tenant.Tenant) (*contact.PaginatedResponse, error) {
			return &contact.PaginatedResponse{
				Meta: contact.PaginationMeta{Total: 1, PerPage: 20, CurrentPage: 1, LastPage: 1, From: 1, To: 1},
				Data: []*contact.Contact{{ID: 1, FirstName: "John", Status: "individual"}},
			}, nil
		},
	}

	app := NewTestApp()
	grp := app.Group("/contacts", TestAuthMiddleware())
	contact.RegisterRoutes(grp, contact.NewContactHandlerWithService(mockSvc))

	req := httptest.NewRequest("GET", "/contacts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp apiResponse
	parseResp(t, resp, &apiResp)
	if apiResp.Meta.Message != "success" {
		t.Errorf("expected message 'success', got %q", apiResp.Meta.Message)
	}
}

func TestContactCreate_201(t *testing.T) {
	now := time.Now()
	mockSvc := &mockContactService{
		createFunc: func(ctx context.Context, req contact.CreateContactRequest, t *tenant.Tenant) (*contact.Contact, error) {
			return &contact.Contact{
				ID: 1, TenantID: t.ID, FirstName: req.FirstName, Status: "individual",
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	app := NewTestApp()
	grp := app.Group("/contacts", TestAuthMiddleware())
	contact.RegisterRoutes(grp, contact.NewContactHandlerWithService(mockSvc))

	body := contact.CreateContactRequest{FirstName: "John"}
	req := httptest.NewRequest("POST", "/contacts", JSONBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var apiResp apiResponse
	parseResp(t, resp, &apiResp)
	if apiResp.Meta.Message != "Contact created" {
		t.Errorf("expected message 'Contact created', got %q", apiResp.Meta.Message)
	}

	var c contact.Contact
	if err := json.Unmarshal(apiResp.Data, &c); err != nil {
		t.Fatal(err)
	}
	if c.FirstName != "John" {
		t.Errorf("expected FirstName 'John', got %q", c.FirstName)
	}
}

func TestContactShow_200(t *testing.T) {
	mockSvc := &mockContactService{
		getByIDFunc: func(ctx context.Context, tenantID int, id int) (*contact.Contact, error) {
			return &contact.Contact{ID: id, TenantID: tenantID, FirstName: "John", Status: "individual"}, nil
		},
	}

	app := NewTestApp()
	grp := app.Group("/contacts", TestAuthMiddleware())
	contact.RegisterRoutes(grp, contact.NewContactHandlerWithService(mockSvc))

	req := httptest.NewRequest("GET", "/contacts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp apiResponse
	parseResp(t, resp, &apiResp)
	if apiResp.Meta.Message != "success" {
		t.Errorf("expected message 'success', got %q", apiResp.Meta.Message)
	}

	var c contact.Contact
	if err := json.Unmarshal(apiResp.Data, &c); err != nil {
		t.Fatal(err)
	}
	if c.ID != 1 || c.FirstName != "John" {
		t.Errorf("unexpected contact: ID=%d, Name=%s", c.ID, c.FirstName)
	}
}

func TestContactUpdate_200(t *testing.T) {
	mockSvc := &mockContactService{
		updateFunc: func(ctx context.Context, tenantID int, id int, req contact.UpdateContactRequest) (*contact.Contact, error) {
			return &contact.Contact{
				ID: id, TenantID: tenantID, FirstName: *req.FirstName, Status: "individual",
			}, nil
		},
	}

	app := NewTestApp()
	grp := app.Group("/contacts", TestAuthMiddleware())
	contact.RegisterRoutes(grp, contact.NewContactHandlerWithService(mockSvc))

	body := contact.UpdateContactRequest{FirstName: strPtr("Jane")}
	req := httptest.NewRequest("PUT", "/contacts/1", JSONBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp apiResponse
	parseResp(t, resp, &apiResp)
	if apiResp.Meta.Message != "Contact updated" {
		t.Errorf("expected message 'Contact updated', got %q", apiResp.Meta.Message)
	}

	var c contact.Contact
	if err := json.Unmarshal(apiResp.Data, &c); err != nil {
		t.Fatal(err)
	}
	if c.FirstName != "Jane" {
		t.Errorf("expected FirstName 'Jane', got %q", c.FirstName)
	}
}

func TestContactDelete_200(t *testing.T) {
	mockSvc := &mockContactService{
		deleteFunc: func(ctx context.Context, tenantID int, id int) error {
			return nil
		},
	}

	app := NewTestApp()
	grp := app.Group("/contacts", TestAuthMiddleware())
	contact.RegisterRoutes(grp, contact.NewContactHandlerWithService(mockSvc))

	req := httptest.NewRequest("DELETE", "/contacts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp apiResponse
	parseResp(t, resp, &apiResp)
	if apiResp.Meta.Message != "Contact deleted" {
		t.Errorf("expected message 'Contact deleted', got %q", apiResp.Meta.Message)
	}
}

func TestContactConversations_200(t *testing.T) {
	mockSvc := &mockContactService{
		getConversationsFunc: func(ctx context.Context, tenantID int, contactID int) (interface{}, error) {
			return []map[string]interface{}{
				{"id": float64(1), "status": "active", "profile_id": float64(10), "channel_id": float64(5), "unread_count": float64(3)},
			}, nil
		},
	}

	app := NewTestApp()
	grp := app.Group("/contacts", TestAuthMiddleware())
	contact.RegisterRoutes(grp, contact.NewContactHandlerWithService(mockSvc))

	req := httptest.NewRequest("GET", "/contacts/1/conversations", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var apiResp apiResponse
	parseResp(t, resp, &apiResp)
	if apiResp.Meta.Message != "success" {
		t.Errorf("expected message 'success', got %q", apiResp.Meta.Message)
	}
}

func TestContactCreate_400(t *testing.T) {
	mockSvc := &mockContactService{}
	app := NewTestApp()
	grp := app.Group("/contacts", TestAuthMiddleware())
	contact.RegisterRoutes(grp, contact.NewContactHandlerWithService(mockSvc))

	req := httptest.NewRequest("POST", "/contacts", strings.NewReader(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
