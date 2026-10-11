package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"centrachannel/internal/middleware"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/user"
)

type mockUserService struct {
	listRolesFn func(context.Context, uuid.UUID) ([]user.Role, error)
	listFn      func(context.Context, user.ListUserQuery, *tenant.Tenant) ([]*user.User, int, error)
	getByIDFn   func(context.Context, uuid.UUID, uuid.UUID) (*user.User, error)
	createFn    func(context.Context, user.CreateUserRequest, *tenant.Tenant) (*user.User, error)
	updateFn    func(context.Context, uuid.UUID, uuid.UUID, user.UpdateUserRequest) (*user.User, error)
	deleteFn    func(context.Context, uuid.UUID, uuid.UUID) error
}

func (m *mockUserService) ListRoles(ctx context.Context, tenantID uuid.UUID) ([]user.Role, error) {
	return m.listRolesFn(ctx, tenantID)
}

func (m *mockUserService) List(ctx context.Context, q user.ListUserQuery, t *tenant.Tenant) ([]*user.User, int, error) {
	return m.listFn(ctx, q, t)
}

func (m *mockUserService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*user.User, error) {
	return m.getByIDFn(ctx, tenantID, id)
}

func (m *mockUserService) Create(ctx context.Context, req user.CreateUserRequest, t *tenant.Tenant) (*user.User, error) {
	return m.createFn(ctx, req, t)
}

func (m *mockUserService) Update(ctx context.Context, tenantID, id uuid.UUID, req user.UpdateUserRequest) (*user.User, error) {
	return m.updateFn(ctx, tenantID, id, req)
}

func (m *mockUserService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return m.deleteFn(ctx, tenantID, id)
}

func newUserApp(svc user.UserService) *fiber.App {
	app := NewTestApp()
	h := user.NewUserHandlerWithService(svc)
	app.Get("/api/users/roles", TestAuthMiddleware(), middleware.Tenant(h.Roles))
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
				{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "John", Username: "john", Email: "john@test.com", CreatedAt: now, UpdatedAt: now},
				{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), FirstName: "Jane", Username: "jane", Email: "jane@test.com", CreatedAt: now, UpdatedAt: now},
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
		Data json.RawMessage `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
		PerPage     int `json:"per_page"`
		CurrentPage int `json:"current_page"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Meta.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Meta.Total)
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
				ID: uuid.MustParse("00000000-0000-4000-8000-000000000003"), TenantID: t.ID, FirstName: req.FirstName,
				Username: req.Username, Email: req.Email,
				Roles:     []user.Role{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "admin"}},
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	app := newUserApp(mock)
	payload := map[string]interface{}{
		"first_name": "New",
		"username":   "newuser",
		"password":   "testpassword123",
		"email":      "new@test.com",
		"roles":      []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000001")},
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
	if err := DecodeData(resp, &u); err != nil {
		t.Fatal(err)
	}
	if u.FirstName != "New" {
		t.Errorf("expected FirstName 'New', got %q", u.FirstName)
	}
}

func TestUserShow(t *testing.T) {
	now := time.Now()
	mock := &mockUserService{
		getByIDFn: func(_ context.Context, tenantID, id uuid.UUID) (*user.User, error) {
			return &user.User{
				ID: id, TenantID: tenantID, FirstName: "John",
				Username: "john", Email: "john@test.com",
				Roles:     []user.Role{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "admin"}},
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	app := newUserApp(mock)
	req, _ := http.NewRequest(http.MethodGet, "/api/users/00000000-0000-4000-8000-000000000001", nil)
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
	if err := DecodeData(resp, &u); err != nil {
		t.Fatal(err)
	}
	if u.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") || u.FirstName != "John" {
		t.Errorf("unexpected user: ID=%s, Name=%s", u.ID, u.FirstName)
	}
}

func TestUserUpdate(t *testing.T) {
	now := time.Now()
	mock := &mockUserService{
		updateFn: func(_ context.Context, tenantID, id uuid.UUID, req user.UpdateUserRequest) (*user.User, error) {
			return &user.User{
				ID: id, TenantID: tenantID, FirstName: req.FirstName,
				Username: req.Username, Email: req.Email,
				Roles:     []user.Role{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "admin"}},
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	app := newUserApp(mock)
	payload := map[string]interface{}{
		"first_name": "Updated",
		"username":   "updateduser",
		"email":      "updated@test.com",
		"roles":      []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000001")},
	}
	req, _ := http.NewRequest(http.MethodPut, "/api/users/00000000-0000-4000-8000-000000000001", JSONBody(payload))
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
	if err := DecodeData(resp, &u); err != nil {
		t.Fatal(err)
	}
	if u.FirstName != "Updated" {
		t.Errorf("expected FirstName 'Updated', got %q", u.FirstName)
	}
}

func TestUserDelete_SuperAdmin(t *testing.T) {
	mock := &mockUserService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return fmt.Errorf("You can not delete super admin")
		},
	}

	app := newUserApp(mock)
	req, _ := http.NewRequest(http.MethodDelete, "/api/users/00000000-0000-4000-8000-000000000001", nil)
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
	if body.Message != "You can not delete super admin" {
		t.Errorf("unexpected message: %q", body.Message)
	}
}

func TestUserDelete_NormalUser(t *testing.T) {
	mock := &mockUserService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return nil
		},
	}

	app := newUserApp(mock)
	req, _ := http.NewRequest(http.MethodDelete, "/api/users/00000000-0000-4000-8000-000000000002", nil)
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

func TestUserRoles(t *testing.T) {
	mock := &mockUserService{listRolesFn: func(_ context.Context, tenantID uuid.UUID) ([]user.Role, error) {
		if tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Fatalf("wrong tenant: %s", tenantID)
		}
		return []user.Role{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), TenantID: tenantID, Name: "agent"}}, nil
	}}
	req, _ := http.NewRequest(http.MethodGet, "/api/users/roles", nil)
	resp, err := newUserApp(mock).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var roles []user.Role
	if err := DecodeData(resp, &roles); err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0].Name != "agent" {
		t.Fatalf("unexpected roles: %+v", roles)
	}
}

func TestUserRequiresRoles(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, roles := range []interface{}{nil, []string{}, []string{"00000000-0000-0000-0000-000000000000"}} {
			t.Run(fmt.Sprintf("%s/%v", method, roles), func(t *testing.T) {
				mock := &mockUserService{}
				path := "/api/users"
				if method == http.MethodPut {
					path += "/00000000-0000-4000-8000-000000000001"
				}
				req, _ := http.NewRequest(method, path, JSONBody(map[string]interface{}{"first_name": "John", "username": "john", "email": "john@test.com", "password": "password123", "roles": roles}))
				req.Header.Set("Content-Type", "application/json")
				resp, err := newUserApp(mock).Test(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusUnprocessableEntity {
					t.Fatalf("expected 422, got %d", resp.StatusCode)
				}
			})
		}
	}
}
