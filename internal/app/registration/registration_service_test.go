package registration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) { return &mockConn{}, nil }

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) { return &mockStmt{}, nil }
func (c *mockConn) Close() error                              { return nil }
func (c *mockConn) Begin() (driver.Tx, error)                 { return &mockTx{}, nil }

type mockTx struct{}

func (t *mockTx) Commit() error   { return nil }
func (t *mockTx) Rollback() error { return nil }

type mockStmt struct{}

func (s *mockStmt) Close() error                                    { return nil }
func (s *mockStmt) NumInput() int                                   { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) { return &mockResult{}, nil }
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error)  { return &mockRows{}, nil }

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 1, nil }
func (r *mockResult) RowsAffected() (int64, error) { return 1, nil }

type mockRows struct{}

func (r *mockRows) Columns() []string              { return nil }
func (r *mockRows) Close() error                   { return nil }
func (r *mockRows) Next(dest []driver.Value) error { return io.EOF }

type mockConnector struct{}

func (c *mockConnector) Connect(ctx context.Context) (driver.Conn, error) { return &mockConn{}, nil }
func (c *mockConnector) Driver() driver.Driver                            { return &mockDriver{} }

type mockTenantRepository struct {
	createFn     func(ctx context.Context, q tenant.DBTX, ten *tenant.Tenant) (uuid.UUID, error)
	createRoleFn func(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error)
	createUserFn func(ctx context.Context, q tenant.DBTX, user *tenant.User) (uuid.UUID, error)
	attachRoleFn func(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error
}

func (m *mockTenantRepository) Create(ctx context.Context, q tenant.DBTX, ten *tenant.Tenant) (uuid.UUID, error) {
	return m.createFn(ctx, q, ten)
}
func (m *mockTenantRepository) CreateRole(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	return m.createRoleFn(ctx, q, tenantID, name)
}
func (m *mockTenantRepository) CreateUser(ctx context.Context, q tenant.DBTX, user *tenant.User) (uuid.UUID, error) {
	return m.createUserFn(ctx, q, user)
}
func (m *mockTenantRepository) AttachRole(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error {
	return m.attachRoleFn(ctx, q, tenantID, userID, roleID)
}
func (m *mockTenantRepository) List(ctx context.Context, q tenant.DBTX) ([]tenant.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByID(ctx context.Context, q tenant.DBTX, id uuid.UUID) (*tenant.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByMetaPageID(ctx context.Context, q tenant.DBTX, pageID string) (*tenant.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByMetaInstagramBusinessID(ctx context.Context, q tenant.DBTX, igID string) (*tenant.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) Update(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) error {
	return nil
}

func newMockDB() *sql.DB {
	return sql.OpenDB(&mockConnector{})
}

func TestOnboard(t *testing.T) {
	t.Run("creates tenant with roles and admin user in transaction", func(t *testing.T) {
		var calls []string
		repo := &mockTenantRepository{
			createFn: func(ctx context.Context, q tenant.DBTX, ten *tenant.Tenant) (uuid.UUID, error) {
				calls = append(calls, "Create")
				if ten.Name != "test-tenant" {
					t.Errorf("unexpected tenant name: %s", ten.Name)
				}
				if !ten.IsActive {
					t.Error("expected tenant to be active")
				}
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
			createRoleFn: func(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
				calls = append(calls, "CreateRole:"+name)
				if tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
					t.Errorf("unexpected tenantID for role %s: %s", name, tenantID)
				}
				switch name {
				case "super-admin":
					return uuid.MustParse("00000000-0000-4000-8000-00000000000a"), nil
				case "admin":
					return uuid.MustParse("00000000-0000-4000-8000-000000000014"), nil
				case "agent":
					return uuid.MustParse("00000000-0000-4000-8000-00000000001e"), nil
				}
				t.Errorf("unexpected role name: %s", name)
				return uuid.Nil, nil
			},
			createUserFn: func(ctx context.Context, q tenant.DBTX, user *tenant.User) (uuid.UUID, error) {
				calls = append(calls, "CreateUser")
				if user.Email != "admin@test.com" {
					t.Errorf("unexpected user email: %s", user.Email)
				}
				if user.TenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
					t.Errorf("unexpected user tenant_id: %s", user.TenantID)
				}
				return uuid.MustParse("00000000-0000-4000-8000-000000000064"), nil
			},
			attachRoleFn: func(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error {
				calls = append(calls, "AttachRole")
				if userID != uuid.MustParse("00000000-0000-4000-8000-000000000064") {
					t.Errorf("unexpected userID for role attach: %s", userID)
				}
				if roleID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
					t.Errorf("expected super-admin role (10), got %s", roleID)
				}
				if tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
					t.Errorf("expected tenantID 1, got %s", tenantID)
				}
				return nil
			},
		}

		svc := NewRegistrationService(repo, newMockDB(), &config.Config{}, logger.NewLogger("debug", "text"))

		resp, err := svc.Onboard(context.Background(), OnboardRequest{
			Name:          "test-tenant",
			Domain:        "test.example.com",
			AdminEmail:    "admin@test.com",
			AdminPassword: "TestPass123!",
		})
		if err != nil {
			t.Fatal(err)
		}

		if resp.Tenant.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected tenant ID 1, got %s", resp.Tenant.ID)
		}
		if resp.Tenant.Name != "test-tenant" {
			t.Errorf("expected tenant name test-tenant, got %s", resp.Tenant.Name)
		}
		if resp.Tenant.Domain != "test.example.com" {
			t.Errorf("expected domain test.example.com, got %s", resp.Tenant.Domain)
		}
		if !resp.Tenant.IsActive {
			t.Error("expected tenant to be active")
		}
		if resp.Admin.ID != uuid.MustParse("00000000-0000-4000-8000-000000000064") {
			t.Errorf("expected admin ID 100, got %s", resp.Admin.ID)
		}
		if resp.Admin.Email != "admin@test.com" {
			t.Errorf("expected admin email admin@test.com, got %s", resp.Admin.Email)
		}
		if resp.Admin.Username != "admin" {
			t.Errorf("expected admin username admin, got %s", resp.Admin.Username)
		}

		expectedCalls := []string{"Create", "CreateRole:super-admin", "CreateRole:admin", "CreateRole:agent", "CreateUser", "AttachRole"}
		if len(calls) != len(expectedCalls) {
			t.Fatalf("expected %d calls, got %d: %v", len(expectedCalls), len(calls), calls)
		}
		for i, call := range expectedCalls {
			if calls[i] != call {
				t.Errorf("call %d: expected %s, got %s", i, call, calls[i])
			}
		}
	})

	t.Run("uses company name as tenant name when provided", func(t *testing.T) {
		repo := &mockTenantRepository{
			createFn: func(ctx context.Context, q tenant.DBTX, ten *tenant.Tenant) (uuid.UUID, error) {
				if ten.Name != "My Corp" {
					t.Errorf("expected tenant name 'My Corp', got %s", ten.Name)
				}
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
			createRoleFn: func(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
				switch name {
				case "super-admin":
					return uuid.MustParse("00000000-0000-4000-8000-00000000000a"), nil
				case "admin":
					return uuid.MustParse("00000000-0000-4000-8000-000000000014"), nil
				case "agent":
					return uuid.MustParse("00000000-0000-4000-8000-00000000001e"), nil
				}
				return uuid.Nil, nil
			},
			createUserFn: func(ctx context.Context, q tenant.DBTX, user *tenant.User) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-4000-8000-000000000064"), nil
			},
			attachRoleFn: func(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error {
				return nil
			},
		}

		svc := NewRegistrationService(repo, newMockDB(), &config.Config{}, logger.NewLogger("debug", "text"))

		resp, err := svc.Onboard(context.Background(), OnboardRequest{
			Name:          "test-tenant",
			CompanyName:   "My Corp",
			Domain:        "corp.example.com",
			AdminEmail:    "admin@corp.com",
			AdminPassword: "TestPass123!",
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Tenant.Name != "My Corp" {
			t.Errorf("expected tenant name 'My Corp', got %s", resp.Tenant.Name)
		}
	})
}
