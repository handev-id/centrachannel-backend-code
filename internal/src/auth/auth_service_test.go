package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock SQL Driver
// ---------------------------------------------------------------------------

type authTestDriver struct{}

func (d *authTestDriver) Open(name string) (driver.Conn, error) {
	return nil, errors.New("use OpenDB")
}

var registeredDriver = &authTestDriver{}

func init() {
	sql.Register("auth-test-mock", registeredDriver)
}

type mockConnector struct {
	conn *mockConn
}

func (c *mockConnector) Connect(_ context.Context) (driver.Conn, error) {
	return c.conn, nil
}

func (c *mockConnector) Driver() driver.Driver {
	return registeredDriver
}

type mockConn struct {
	prepareFn func(query string) (driver.Stmt, error)
	beginFn   func() (driver.Tx, error)
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	if c.prepareFn != nil {
		return c.prepareFn(query)
	}
	return &mockStmt{query: query}, nil
}

func (c *mockConn) Close() error { return nil }

func (c *mockConn) Begin() (driver.Tx, error) {
	if c.beginFn != nil {
		return c.beginFn()
	}
	return &mockTx{}, nil
}

type mockTx struct {
	commitFn   func() error
	rollbackFn func() error
}

func (t *mockTx) Commit() error {
	if t.commitFn != nil {
		return t.commitFn()
	}
	return nil
}

func (t *mockTx) Rollback() error {
	if t.rollbackFn != nil {
		return t.rollbackFn()
	}
	return nil
}

type mockStmt struct {
	query   string
	execFn  func(args []driver.Value) (driver.Result, error)
	queryFn func(args []driver.Value) (driver.Rows, error)
}

func (s *mockStmt) Close() error { return nil }

func (s *mockStmt) NumInput() int { return -1 }

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.execFn != nil {
		return s.execFn(args)
	}
	return &mockResult{insertID: 1, affected: 1}, nil
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.queryFn != nil {
		return s.queryFn(args)
	}
	return &mockRows{
		cols: []string{"id"},
		data: [][]driver.Value{{int64(1)}},
	}, nil
}

type mockRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (r *mockRows) Columns() []string { return r.cols }

func (r *mockRows) Close() error { return nil }

func (r *mockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

type mockResult struct {
	insertID int64
	affected int64
}

func (r *mockResult) LastInsertId() (int64, error) { return r.insertID, nil }

func (r *mockResult) RowsAffected() (int64, error) { return r.affected, nil }

func openMockDB(conn *mockConn) *sql.DB {
	db := sql.OpenDB(&mockConnector{conn: conn})
	db.SetMaxOpenConns(1)
	return db
}

// ---------------------------------------------------------------------------
// Mock Auth Repository
// ---------------------------------------------------------------------------

type mockAuthRepository struct {
	getByEmailFn        func(ctx context.Context, q DBTX, tenantID int, email string) (*User, error)
	getByUsernameFn     func(ctx context.Context, q DBTX, tenantID int, username string) (*User, error)
	getByIDFn           func(ctx context.Context, q DBTX, tenantID int, id int) (*User, error)
	createFn            func(ctx context.Context, q DBTX, user *User) (int, error)
	getRolesByUserIDFn  func(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error)
}

func (m *mockAuthRepository) GetByEmail(ctx context.Context, q DBTX, tenantID int, email string) (*User, error) {
	if m.getByEmailFn == nil {
		return nil, nil
	}
	return m.getByEmailFn(ctx, q, tenantID, email)
}

func (m *mockAuthRepository) GetByUsername(ctx context.Context, q DBTX, tenantID int, username string) (*User, error) {
	if m.getByUsernameFn == nil {
		return nil, nil
	}
	return m.getByUsernameFn(ctx, q, tenantID, username)
}

func (m *mockAuthRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*User, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, q, tenantID, id)
}

func (m *mockAuthRepository) Create(ctx context.Context, q DBTX, user *User) (int, error) {
	if m.createFn == nil {
		return 0, nil
	}
	return m.createFn(ctx, q, user)
}

func (m *mockAuthRepository) GetRolesByUserID(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error) {
	if m.getRolesByUserIDFn == nil {
		return nil, nil
	}
	return m.getRolesByUserIDFn(ctx, q, tenantID, userID)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret: "test-secret",
		JWTExpiry: 24 * time.Hour,
	}
}

func testLogger() *logger.Logger {
	return logger.NewLogger("error", "text")
}

func testTenant() *tenant.Tenant {
	return &tenant.Tenant{ID: 1, Domain: "test.domain"}
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister(t *testing.T) {
	cfg := testConfig()
	log := testLogger()
	ten := testTenant()

	t.Run("duplicate email returns error", func(t *testing.T) {
		repo := &mockAuthRepository{
			getByEmailFn: func(_ context.Context, _ DBTX, _ int, email string) (*User, error) {
				return &User{ID: 1, Email: email}, nil
			},
		}
		svc := NewAuthService(repo, openMockDB(&mockConn{}), cfg, log)

		_, err := svc.Register(context.Background(), RegisterRequest{Email: "dup@test.com", Username: "u"}, ten)
		if err == nil || err.Error() != "email already registered" {
			t.Fatalf("expected 'email already registered', got %v", err)
		}
	})

	t.Run("duplicate username returns error", func(t *testing.T) {
		repo := &mockAuthRepository{
			getByEmailFn: func(_ context.Context, _ DBTX, _ int, _ string) (*User, error) {
				return nil, nil
			},
			getByUsernameFn: func(_ context.Context, _ DBTX, _ int, username string) (*User, error) {
				return &User{ID: 2, Username: username}, nil
			},
		}
		svc := NewAuthService(repo, openMockDB(&mockConn{}), cfg, log)

		_, err := svc.Register(context.Background(), RegisterRequest{Email: "ok@test.com", Username: "taken"}, ten)
		if err == nil || err.Error() != "username already taken" {
			t.Fatalf("expected 'username already taken', got %v", err)
		}
	})

	t.Run("successful registration", func(t *testing.T) {
		repo := &mockAuthRepository{
			getByEmailFn: func(_ context.Context, _ DBTX, _ int, _ string) (*User, error) {
				return nil, nil
			},
			getByUsernameFn: func(_ context.Context, _ DBTX, _ int, _ string) (*User, error) {
				return nil, nil
			},
			createFn: func(_ context.Context, _ DBTX, user *User) (int, error) {
				if user.Password == "" {
					t.Error("expected non-empty hashed password")
				}
				if user.TenantID != 1 {
					t.Errorf("expected tenant ID 1, got %d", user.TenantID)
				}
				return 42, nil
			},
		}

		conn := &mockConn{}
		db := openMockDB(conn)
		t.Cleanup(func() { db.Close() })

		svc := NewAuthService(repo, db, cfg, log)

		password := "myp@ssword1"
		req := RegisterRequest{
			FirstName: "John",
			Username:  "john",
			Email:     "john@test.com",
			Password:  &password,
		}

		user, err := svc.Register(context.Background(), req, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != 42 {
			t.Errorf("expected user ID 42, got %d", user.ID)
		}
		if user.Username != "john" {
			t.Errorf("expected username 'john', got %s", user.Username)
		}
		if len(user.Roles) != 1 || user.Roles[0].ID != 1 {
			t.Errorf("expected role with ID 1, got %+v", user.Roles)
		}
		if user.Roles[0].Name != "agent" {
			t.Errorf("expected role name 'agent', got %s", user.Roles[0].Name)
		}
	})
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin(t *testing.T) {
	cfg := testConfig()
	log := testLogger()
	ten := testTenant()

	// Pre-compute a bcrypt hash so we can match/fail passwords
	hashedPW, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		req     LoginRequest
		mockFn  func() *mockAuthRepository
		wantErr bool
	}{
		{
			name: "successful login",
			req:  LoginRequest{Username: "testuser", Password: "correctpassword"},
			mockFn: func() *mockAuthRepository {
				return &mockAuthRepository{
					getByUsernameFn: func(_ context.Context, _ DBTX, _ int, _ string) (*User, error) {
						return &User{ID: 1, TenantID: 1, Username: "testuser", Password: string(hashedPW)}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "invalid credentials - wrong password",
			req:  LoginRequest{Username: "testuser", Password: "wrongpassword"},
			mockFn: func() *mockAuthRepository {
				return &mockAuthRepository{
					getByUsernameFn: func(_ context.Context, _ DBTX, _ int, _ string) (*User, error) {
						return &User{ID: 1, TenantID: 1, Username: "testuser", Password: string(hashedPW)}, nil
					},
				}
			},
			wantErr: true,
		},
		{
			name: "invalid credentials - user not found",
			req:  LoginRequest{Username: "nobody", Password: "doesnotmatter"},
			mockFn: func() *mockAuthRepository {
				return &mockAuthRepository{
					getByUsernameFn: func(_ context.Context, _ DBTX, _ int, _ string) (*User, error) {
						return nil, nil
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openMockDB(&mockConn{})
			t.Cleanup(func() { db.Close() })

			svc := NewAuthService(tt.mockFn(), db, cfg, log)
			token, err := svc.Login(context.Background(), tt.req, ten)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if token != "" {
					t.Error("expected empty token on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token == "" {
				t.Fatal("expected non-empty token")
			}

			// Verify the returned token is valid and contains the right claims
			parsed, err := jwt.Parse(token, func(tk *jwt.Token) (interface{}, error) {
				if _, ok := tk.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil {
				t.Fatalf("token should be parseable: %v", err)
			}
			claims, ok := parsed.Claims.(jwt.MapClaims)
			if !ok {
				t.Fatal("expected MapClaims")
			}
			if got := claims["sub"]; got != float64(1) {
				t.Errorf("expected sub=1, got %v", got)
			}
			if got := claims["tenant"]; got != float64(1) {
				t.Errorf("expected tenant=1, got %v", got)
			}
			if got := claims["domain"]; got != "test.domain" {
				t.Errorf("expected domain=test.domain, got %v", got)
			}
			if got := claims["user"]; got != "testuser" {
				t.Errorf("expected user=testuser, got %v", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CheckToken
// ---------------------------------------------------------------------------

func TestCheckToken(t *testing.T) {
	cfg := testConfig()
	log := testLogger()
	ten := testTenant()

	t.Run("valid token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":    float64(1),
			"tenant": float64(1),
			"domain": "test.domain",
			"exp":    float64(time.Now().Add(time.Hour).Unix()),
			"user":   "testuser",
		})
		tokenStr, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatal(err)
		}

		repo := &mockAuthRepository{
			getByIDFn: func(_ context.Context, _ DBTX, _ int, id int) (*User, error) {
				return &User{ID: id, TenantID: 1, Username: "testuser"}, nil
			},
			getRolesByUserIDFn: func(_ context.Context, _ DBTX, _ int, _ int) ([]Role, error) {
				return []Role{{ID: 1, TenantID: 1, Name: "agent"}}, nil
			},
		}

		db := openMockDB(&mockConn{})
		t.Cleanup(func() { db.Close() })

		svc := NewAuthService(repo, db, cfg, log)

		user, err := svc.CheckToken(context.Background(), tokenStr, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != 1 {
			t.Errorf("expected user ID 1, got %d", user.ID)
		}
		if user.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %s", user.Username)
		}
		if len(user.Roles) != 1 || user.Roles[0].Name != "agent" {
			t.Errorf("expected role 'agent', got %+v", user.Roles)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": float64(1),
			"exp": float64(time.Now().Add(-time.Hour).Unix()),
		})
		tokenStr, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatal(err)
		}

		db := openMockDB(&mockConn{})
		t.Cleanup(func() { db.Close() })

		svc := NewAuthService(&mockAuthRepository{}, db, cfg, log)

		_, err = svc.CheckToken(context.Background(), tokenStr, ten)
		if err == nil || err.Error() != "invalid token" {
			t.Fatalf("expected 'invalid token', got %v", err)
		}
	})

	t.Run("invalid signing method", func(t *testing.T) {
		// Create a token signed with RS256 using a throwaway key pair.
		// The keyFunc checks tk.Method.(*jwt.SigningMethodHMAC) which will
		// be false for RS256, causing it to return "unexpected signing method".
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": float64(1),
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		})
		tokenStr, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}

		db := openMockDB(&mockConn{})
		t.Cleanup(func() { db.Close() })

		svc := NewAuthService(&mockAuthRepository{}, db, cfg, log)

		_, err = svc.CheckToken(context.Background(), tokenStr, ten)
		if err == nil || err.Error() != "invalid token" {
			t.Fatalf("expected 'invalid token', got %v", err)
		}
	})
}
