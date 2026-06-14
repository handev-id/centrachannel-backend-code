package integration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"centrachannel/internal/src/auth"
	"centrachannel/internal/src/tenant"
)

type mockAuthService struct {
	registerFn   func(ctx context.Context, req auth.RegisterRequest, t *tenant.Tenant) (*auth.User, error)
	loginFn      func(ctx context.Context, req auth.LoginRequest, t *tenant.Tenant) (string, error)
	checkTokenFn func(ctx context.Context, token string, t *tenant.Tenant) (*auth.User, error)
}

func (m *mockAuthService) Register(ctx context.Context, req auth.RegisterRequest, t *tenant.Tenant) (*auth.User, error) {
	return m.registerFn(ctx, req, t)
}

func (m *mockAuthService) Login(ctx context.Context, req auth.LoginRequest, t *tenant.Tenant) (string, error) {
	return m.loginFn(ctx, req, t)
}

func (m *mockAuthService) CheckToken(ctx context.Context, token string, t *tenant.Tenant) (*auth.User, error) {
	return m.checkTokenFn(ctx, token, t)
}

type metaJSON struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type apiJSON struct {
	Meta   metaJSON    `json:"meta"`
	Data   interface{} `json:"data,omitempty"`
	Errors interface{} `json:"errors,omitempty"`
}

func readResponse(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("failed to decode response body: %v\nbody: %s", err, string(body))
	}
}

func TestAuthRegister_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		registerFn: func(ctx context.Context, req auth.RegisterRequest, t *tenant.Tenant) (*auth.User, error) {
			return &auth.User{
				ID:        1,
				TenantID:  t.ID,
				FirstName: req.FirstName,
				Username:  req.Username,
				Email:     req.Email,
				Roles:     []auth.Role{{ID: 1, TenantID: t.ID, Name: "agent"}},
			}, nil
		},
	}

	app := NewTestApp()
	handler := auth.NewAuthHandlerWithService(mockSvc)
	app.Post("/api/auth/register", TestAuthMiddleware(), handler.Register)

	body := JSONBody(auth.RegisterRequest{
		FirstName: "John",
		Username:  "john_doe",
		Email:     "john@test.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var apiResp apiJSON
	readResponse(t, resp, &apiResp)

	if apiResp.Meta.Code != 201 {
		t.Errorf("meta.code = %d, want 201", apiResp.Meta.Code)
	}
	if apiResp.Meta.Message != "User registered" {
		t.Errorf("meta.message = %q, want %q", apiResp.Meta.Message, "User registered")
	}
	if apiResp.Data == nil {
		t.Error("data is nil, expected user object")
	}

	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data type = %T, want map[string]interface{}", apiResp.Data)
	}
	if data["id"].(float64) != 1 {
		t.Errorf("data.id = %v, want 1", data["id"])
	}
	if data["username"] != "john_doe" {
		t.Errorf("data.username = %v, want john_doe", data["username"])
	}
	if data["email"] != "john@test.com" {
		t.Errorf("data.email = %v, want john@test.com", data["email"])
	}
}

func TestAuthRegister_DuplicateEmail(t *testing.T) {
	mockSvc := &mockAuthService{
		registerFn: func(ctx context.Context, req auth.RegisterRequest, t *tenant.Tenant) (*auth.User, error) {
			return nil, errors.New("email already registered")
		},
	}

	app := NewTestApp()
	handler := auth.NewAuthHandlerWithService(mockSvc)
	app.Post("/api/auth/register", TestAuthMiddleware(), handler.Register)

	body := JSONBody(auth.RegisterRequest{
		FirstName: "Jane",
		Username:  "jane_doe",
		Email:     "existing@test.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var apiResp apiJSON
	readResponse(t, resp, &apiResp)

	if apiResp.Meta.Code != 400 {
		t.Errorf("meta.code = %d, want 400", apiResp.Meta.Code)
	}
}

func TestAuthLogin_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		loginFn: func(ctx context.Context, req auth.LoginRequest, t *tenant.Tenant) (string, error) {
			return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token", nil
		},
	}

	app := NewTestApp()
	handler := auth.NewAuthHandlerWithService(mockSvc)
	app.Post("/api/auth/login", TestAuthMiddleware(), handler.Login)

	body := JSONBody(auth.LoginRequest{
		Username: "john_doe",
		Password: "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var apiResp apiJSON
	readResponse(t, resp, &apiResp)

	if apiResp.Meta.Code != 200 {
		t.Errorf("meta.code = %d, want 200", apiResp.Meta.Code)
	}
	if apiResp.Meta.Message != "Login successful" {
		t.Errorf("meta.message = %q, want %q", apiResp.Meta.Message, "Login successful")
	}

	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data type = %T, want map[string]interface{}", apiResp.Data)
	}
	if data["type"] != "bearer" {
		t.Errorf("data.type = %v, want bearer", data["type"])
	}
	if data["token"] != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token" {
		t.Errorf("data.token = %v, want %v", data["token"], "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token")
	}
}

func TestAuthLogin_InvalidCredentials(t *testing.T) {
	mockSvc := &mockAuthService{
		loginFn: func(ctx context.Context, req auth.LoginRequest, t *tenant.Tenant) (string, error) {
			return "", errors.New("invalid credentials")
		},
	}

	app := NewTestApp()
	handler := auth.NewAuthHandlerWithService(mockSvc)
	app.Post("/api/auth/login", TestAuthMiddleware(), handler.Login)

	body := JSONBody(auth.LoginRequest{
		Username: "wrong",
		Password: "wrongpass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	var apiResp apiJSON
	readResponse(t, resp, &apiResp)

	if apiResp.Meta.Code != 401 {
		t.Errorf("meta.code = %d, want 401", apiResp.Meta.Code)
	}
	if apiResp.Meta.Message != "invalid credentials" {
		t.Errorf("meta.message = %q, want %q", apiResp.Meta.Message, "invalid credentials")
	}
}

func TestAuthCheckToken_Valid(t *testing.T) {
	mockSvc := &mockAuthService{
		checkTokenFn: func(ctx context.Context, token string, t *tenant.Tenant) (*auth.User, error) {
			return &auth.User{
				ID:        1,
				TenantID:  t.ID,
				FirstName: "John",
				Username:  "john_doe",
				Email:     "john@test.com",
				Roles:     []auth.Role{{ID: 1, TenantID: t.ID, Name: "agent"}},
			}, nil
		},
	}

	app := NewTestApp()
	handler := auth.NewAuthHandlerWithService(mockSvc)
	app.Get("/api/auth/check-token", TestAuthMiddleware(), handler.CheckToken)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/check-token", nil)
	req.Header.Set("Authorization", "Bearer valid_token")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var apiResp apiJSON
	readResponse(t, resp, &apiResp)

	if apiResp.Meta.Code != 200 {
		t.Errorf("meta.code = %d, want 200", apiResp.Meta.Code)
	}
	if apiResp.Meta.Message != "Token valid" {
		t.Errorf("meta.message = %q, want %q", apiResp.Meta.Message, "Token valid")
	}

	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data type = %T, want map[string]interface{}", apiResp.Data)
	}
	if data["id"].(float64) != 1 {
		t.Errorf("data.id = %v, want 1", data["id"])
	}
	if data["username"] != "john_doe" {
		t.Errorf("data.username = %v, want john_doe", data["username"])
	}
}
