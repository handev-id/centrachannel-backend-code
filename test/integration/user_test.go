package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/user"
)

type mockUserService struct {
	listFn    func(context.Context, user.ListUserQuery, *tenant.Tenant) (*user.PaginatedResponse, error)
	getByIDFn func(context.Context, int, int) (*user.User, error)
	createFn  func(context.Context, user.CreateUserRequest, *tenant.Tenant) (*user.User, error)
	updateFn  func(context.Context, int, int, user.UpdateUserRequest) (*user.User, error)
	deleteFn  func(context.Context, int, int) error
}

func (m *mockUserService) List(ctx context.Context, q user.ListUserQuery, t *tenant.Tenant) (*user.PaginatedResponse, error) {
	return m.listFn(ctx, q, t)
}

func (m *mockUserService) GetByID(ctx context.Context, tenantID, id int) (*user.User, error) {
	return m.getByIDFn(ctx, tenantID, id)
}

func (m *mockUserService) Create(ctx context.Context, req user.CreateUserRequest, t *tenant.Tenant) (*user.User, error) {
	return m.createFn(ctx, req, t)
}

func (m *mockUserService) Update(ctx context.Context, tenantID, id int, req user.UpdateUserRequest) (*user.User, error) {
	return m.updateFn(ctx, tenantID, id, req)
}

func (m *mockUserService) Delete(ctx context.Context, tenantID, id int) error {
	return m.deleteFn(ctx, tenantID, id)
}

type testResp struct {
	Meta struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"meta"`
	Data   json.RawMessage `json:"data,omitempty"`
	Errors json.RawMessage `json:"errors,omitempty"`
}

func newUserApp(svc user.UserService) *fiber.App {
	app := NewTestApp()
	h := user.NewUserHandlerWithService(svc)
	app.Get("/api/users", TestAuthMiddleware(), middleware.Tenant(h.List))
	app.Post("/api/users", TestAuthMiddleware(), middleware.Tenant(h.Store))
	app.Get("/api/users/:id", TestAuthMiddleware(), middleware.Tenant(h.Show))
	app.Put("/api/users/:id", TestAuthMiddleware(), middleware.Tenant(h.Update))
	app.Delete("/api/users/:id", TestAuthMiddleware(), middleware.Tenant(h.Delete))
	return app
}

func decodeResp(t *testing.T, resp *http.Response) testResp {
	t.Helper()
	var r testResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestUserList(t *testing.T) {
	now := time.Now()
	mock := &mockUserService{
		listFn: func(_ context.Context, _ user.ListUserQuery, _ *tenant.Tenant) (*user.PaginatedResponse, error) {
			return &user.PaginatedResponse{
				Meta: user.PaginationMeta{Total: 2, PerPage: 20, CurrentPage: 1, LastPage: 1, From: 1, To: 2},
				Data: []user.User{
					{ID: 1, FirstName: "John", Username: "john", Email: "john@test.com", CreatedAt: now, UpdatedAt: now},
					{ID: 2, FirstName: "Jane", Username: "jane", Email: "jane@test.com", CreatedAt: now, UpdatedAt: now},
				},
			}, nil
		},
	}

	app := newUserApp(mock)
	req, _ := http.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	r := decodeResp(t, resp)
	if r.Meta.Code != 200 || r.Meta.Message != "success" {
		t.Errorf("unexpected meta: code=%d message=%s", r.Meta.Code, r.Meta.Message)
	}
	if len(r.Data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestUserCreate(t *testing.T) {
	now := time.Now()
	mock := &mockUserService{
		createFn: func(_ context.Context, req user.CreateUserRequest, t *tenant.Tenant) (*user.User, error) {
			return &user.User{
				ID: 3, TenantID: t.ID, FirstName: req.FirstName,
				Username: req.Username, Email: req.Email,
				Roles:     []user.Role{{ID: 1, Name: "admin"}},
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	app := newUserApp(mock)
	payload := map[string]interface{}{
		"first_name": "New",
		"username":   "newuser",
		"email":      "new@test.com",
		"roles":      []int{1},
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/users", JSONBody(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	r := decodeResp(t, resp)
	if r.Meta.Code != 201 || r.Meta.Message != "User created" {
		t.Errorf("unexpected meta: code=%d message=%s", r.Meta.Code, r.Meta.Message)
	}
}

func TestUserShow(t *testing.T) {
	now := time.Now()
	mock := &mockUserService{
		getByIDFn: func(_ context.Context, tenantID, id int) (*user.User, error) {
			return &user.User{
				ID: id, TenantID: tenantID, FirstName: "John",
				Username: "john", Email: "john@test.com",
				Roles:     []user.Role{{ID: 1, Name: "admin"}},
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	app := newUserApp(mock)
	req, _ := http.NewRequest(http.MethodGet, "/api/users/1", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	r := decodeResp(t, resp)
	if r.Meta.Code != 200 || r.Meta.Message != "success" {
		t.Errorf("unexpected meta: code=%d message=%s", r.Meta.Code, r.Meta.Message)
	}
}

func TestUserUpdate(t *testing.T) {
	now := time.Now()
	mock := &mockUserService{
		updateFn: func(_ context.Context, tenantID, id int, req user.UpdateUserRequest) (*user.User, error) {
			return &user.User{
				ID: id, TenantID: tenantID, FirstName: req.FirstName,
				Username: req.Username, Email: req.Email,
				Roles:     []user.Role{{ID: 1, Name: "admin"}},
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	app := newUserApp(mock)
	payload := map[string]interface{}{
		"first_name": "Updated",
		"username":   "updateduser",
		"email":      "updated@test.com",
		"roles":      []int{1},
	}
	req, _ := http.NewRequest(http.MethodPut, "/api/users/1", JSONBody(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	r := decodeResp(t, resp)
	if r.Meta.Code != 200 || r.Meta.Message != "User updated" {
		t.Errorf("unexpected meta: code=%d message=%s", r.Meta.Code, r.Meta.Message)
	}
}

func TestUserDelete_SuperAdmin(t *testing.T) {
	mock := &mockUserService{
		deleteFn: func(_ context.Context, _, _ int) error {
			return fmt.Errorf("forbidden: you can not delete super admin")
		},
	}

	app := newUserApp(mock)
	req, _ := http.NewRequest(http.MethodDelete, "/api/users/1", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}

	r := decodeResp(t, resp)
	if r.Meta.Code != 403 || r.Meta.Message != "forbidden: you can not delete super admin" {
		t.Errorf("unexpected meta: code=%d message=%s", r.Meta.Code, r.Meta.Message)
	}
}

func TestUserDelete_NormalUser(t *testing.T) {
	mock := &mockUserService{
		deleteFn: func(_ context.Context, _, _ int) error {
			return nil
		},
	}

	app := newUserApp(mock)
	req, _ := http.NewRequest(http.MethodDelete, "/api/users/2", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	r := decodeResp(t, resp)
	if r.Meta.Code != 200 || r.Meta.Message != "User deleted successfully" {
		t.Errorf("unexpected meta: code=%d message=%s", r.Meta.Code, r.Meta.Message)
	}
}
