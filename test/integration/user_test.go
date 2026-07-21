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
	listFn    func(context.Context, user.ListUserQuery, *tenant.Tenant) ([]*user.User, int, error)
	getByIDFn func(context.Context, int, int) (*user.User, error)
	createFn  func(context.Context, user.CreateUserRequest, *tenant.Tenant) (*user.User, error)
	updateFn  func(context.Context, int, int, user.UpdateUserRequest) (*user.User, error)
	deleteFn  func(context.Context, int, int) error
}

func (m *mockUserService) List(ctx context.Context, q user.ListUserQuery, t *tenant.Tenant) ([]*user.User, int, error) {
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

func TestUserList(t *testing.T) {
	now := time.Now()
	mock := &mockUserService{
		listFn: func(_ context.Context, _ user.ListUserQuery, _ *tenant.Tenant) ([]*user.User, int, error) {
			return []*user.User{
				{ID: 1, FirstName: "John", Username: "john", Email: "john@test.com", CreatedAt: now, UpdatedAt: now},
				{ID: 2, FirstName: "Jane", Username: "jane", Email: "jane@test.com", CreatedAt: now, UpdatedAt: now},
			}, 2, nil
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

	var result struct {
		Data        json.RawMessage `json:"data"`
		Total       int             `json:"total"`
		PerPage     int             `json:"per_page"`
		CurrentPage int             `json:"current_page"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if len(result.Data) == 0 {
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

	var u user.User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		t.Fatal(err)
	}
	if u.FirstName != "New" {
		t.Errorf("expected FirstName 'New', got %q", u.FirstName)
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

	var u user.User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 || u.FirstName != "John" {
		t.Errorf("unexpected user: ID=%d, Name=%s", u.ID, u.FirstName)
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

	var u user.User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		t.Fatal(err)
	}
	if u.FirstName != "Updated" {
		t.Errorf("expected FirstName 'Updated', got %q", u.FirstName)
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

	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "forbidden: you can not delete super admin" {
		t.Errorf("unexpected message: %q", body.Message)
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
}
