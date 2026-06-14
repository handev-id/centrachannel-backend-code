package tenant

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	"centrachannel/config"
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

func (s *mockStmt) Close() error                                      { return nil }
func (s *mockStmt) NumInput() int                                      { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error)   { return &mockResult{}, nil }
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error)    { return &mockRows{}, nil }

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
	listFn func(ctx context.Context, q DBTX) ([]Tenant, error)
}

func (m *mockTenantRepository) Create(ctx context.Context, q DBTX, tenant *Tenant) (int, error) {
	return 0, nil
}
func (m *mockTenantRepository) CreateRole(ctx context.Context, q DBTX, tenantID int, name string) (int, error) {
	return 0, nil
}
func (m *mockTenantRepository) CreateUser(ctx context.Context, q DBTX, user *User) (int, error) {
	return 0, nil
}
func (m *mockTenantRepository) AttachRole(ctx context.Context, q DBTX, tenantID, userID, roleID int) error {
	return nil
}
func (m *mockTenantRepository) List(ctx context.Context, q DBTX) ([]Tenant, error) {
	return m.listFn(ctx, q)
}
func (m *mockTenantRepository) GetByID(ctx context.Context, q DBTX, id int) (*Tenant, error) {
	return &Tenant{}, nil
}
func (m *mockTenantRepository) GetByMetaPageID(ctx context.Context, q DBTX, pageID string) (*Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByMetaInstagramBusinessID(ctx context.Context, q DBTX, igID string) (*Tenant, error) {
	return nil, nil
}

func newMockDB() *sql.DB {
	return sql.OpenDB(&mockConnector{})
}

func TestList(t *testing.T) {
	t.Run("returns all tenants", func(t *testing.T) {
		expected := []Tenant{
			{ID: 1, Name: "Alpha", Domain: "alpha.com", IsActive: true},
			{ID: 2, Name: "Beta", Domain: "beta.com", IsActive: true},
		}

		repo := &mockTenantRepository{
			listFn: func(ctx context.Context, q DBTX) ([]Tenant, error) {
				return expected, nil
			},
		}

		svc := NewTenantService(repo, newMockDB(), &config.Config{}, logger.NewLogger("debug", "text"))

		tenants, err := svc.List(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(tenants) != 2 {
			t.Fatalf("expected 2 tenants, got %d", len(tenants))
		}
		if tenants[0].Name != "Alpha" {
			t.Errorf("expected tenant name Alpha, got %s", tenants[0].Name)
		}
		if tenants[1].Name != "Beta" {
			t.Errorf("expected tenant name Beta, got %s", tenants[1].Name)
		}
	})
}
