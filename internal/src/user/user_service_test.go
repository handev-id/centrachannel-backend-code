package user

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

// ---------------------------------------------------------------------------
// Mock database driver (pure stdlib, no external dependencies)
// ---------------------------------------------------------------------------

type mockDriverTx struct {
	committed  bool
	rolledBack bool
}

func (t *mockDriverTx) Commit() error   { t.committed = true; return nil }
func (t *mockDriverTx) Rollback() error { t.rolledBack = true; return nil }

type mockRows struct{}

func (r *mockRows) Columns() []string                  { return nil }
func (r *mockRows) Close() error                       { return nil }
func (r *mockRows) Next(dest []driver.Value) error      { return io.EOF }

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (r *mockResult) RowsAffected() (int64, error) { return 0, nil }

type mockStmt struct{}

func (s *mockStmt) Close() error                                    { return nil }
func (s *mockStmt) NumInput() int                                   { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) { return &mockResult{}, nil }
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error)  { return &mockRows{}, nil }

type mockConn struct {
	tx *mockDriverTx
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) { return &mockStmt{}, nil }
func (c *mockConn) Close() error                              { return nil }
func (c *mockConn) Begin() (driver.Tx, error) {
	c.tx = &mockDriverTx{}
	return c.tx, nil
}

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

func init() {
	sql.Register("user_test_mock", &mockDriver{})
}

func newMockDB() *sql.DB {
	db, err := sql.Open("user_test_mock", "")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	return db
}

// ---------------------------------------------------------------------------
// Mock UserRepository with call capture
// ---------------------------------------------------------------------------

type mockUserRepository struct {
	listFunc              func(context.Context, DBTX, int, int, int, string, *int) ([]*User, int, error)
	getByIDFunc           func(context.Context, DBTX, int, int) (*User, error)
	createFunc            func(context.Context, DBTX, *User) (int, error)
	updateFunc            func(context.Context, DBTX, int, int, *User) error
	softDeleteFunc        func(context.Context, DBTX, int, int) error
	getRolesByUserIDFunc  func(context.Context, DBTX, int, int) ([]Role, error)
	getRolesByUserIDsFunc func(context.Context, DBTX, int, []int) (map[int][]Role, error)
	attachRolesFunc       func(context.Context, DBTX, int, int, []int) error
	syncRolesFunc         func(context.Context, DBTX, int, int, []int) error

	createCalled         bool
	createUser           *User
	attachRolesCalled    bool
	attachRolesUserID    int
	attachRolesIDs       []int
	updateCalled         bool
	updateUser           *User
	syncRolesCalled      bool
	syncRolesUserID      int
	syncRolesIDs         []int
	softDeleteCalled     bool
	softDeleteTenantID   int
	softDeleteID         int
	listCalled           bool
	listLimit            int
	listOffset           int
	getRolesByUserIDsCalled bool
	getRolesByUserIDs    []int
}

func (m *mockUserRepository) List(ctx context.Context, q DBTX, tenantID int, limit, offset int, search string, roleID *int) ([]*User, int, error) {
	m.listCalled = true
	m.listLimit = limit
	m.listOffset = offset
	if m.listFunc != nil {
		return m.listFunc(ctx, q, tenantID, limit, offset, search, roleID)
	}
	return nil, 0, nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, tenantID, id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepository) Create(ctx context.Context, q DBTX, user *User) (int, error) {
	m.createCalled = true
	m.createUser = user
	if m.createFunc != nil {
		return m.createFunc(ctx, q, user)
	}
	return 0, nil
}

func (m *mockUserRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, user *User) error {
	m.updateCalled = true
	m.updateUser = user
	if m.updateFunc != nil {
		return m.updateFunc(ctx, q, tenantID, id, user)
	}
	return nil
}

func (m *mockUserRepository) SoftDelete(ctx context.Context, q DBTX, tenantID int, id int) error {
	m.softDeleteCalled = true
	m.softDeleteTenantID = tenantID
	m.softDeleteID = id
	if m.softDeleteFunc != nil {
		return m.softDeleteFunc(ctx, q, tenantID, id)
	}
	return nil
}

func (m *mockUserRepository) GetRolesByUserID(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error) {
	if m.getRolesByUserIDFunc != nil {
		return m.getRolesByUserIDFunc(ctx, q, tenantID, userID)
	}
	return nil, nil
}

func (m *mockUserRepository) GetRolesByUserIDs(ctx context.Context, q DBTX, tenantID int, userIDs []int) (map[int][]Role, error) {
	m.getRolesByUserIDsCalled = true
	m.getRolesByUserIDs = userIDs
	if m.getRolesByUserIDsFunc != nil {
		return m.getRolesByUserIDsFunc(ctx, q, tenantID, userIDs)
	}
	return map[int][]Role{}, nil
}

func (m *mockUserRepository) AttachRoles(ctx context.Context, q DBTX, tenantID int, userID int, roleIDs []int) error {
	m.attachRolesCalled = true
	m.attachRolesUserID = userID
	m.attachRolesIDs = roleIDs
	if m.attachRolesFunc != nil {
		return m.attachRolesFunc(ctx, q, tenantID, userID, roleIDs)
	}
	return nil
}

func (m *mockUserRepository) SyncRoles(ctx context.Context, q DBTX, tenantID int, userID int, roleIDs []int) error {
	m.syncRolesCalled = true
	m.syncRolesUserID = userID
	m.syncRolesIDs = roleIDs
	if m.syncRolesFunc != nil {
		return m.syncRolesFunc(ctx, q, tenantID, userID, roleIDs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestService(repo *mockUserRepository) *userService {
	return &userService{
		repo:   repo,
		db:     newMockDB(),
		cfg:    &config.Config{},
		logger: logger.NewLogger("error", "text"),
	}
}

func ptr(s string) *string { return &s }

var testTenant = &tenant.Tenant{ID: 1, Name: "Test Tenant"}

// ---------------------------------------------------------------------------
// Tests — Create
// ---------------------------------------------------------------------------

func TestCreate(t *testing.T) {
	t.Run("generates dicebear avatar when none provided", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		req := CreateUserRequest{
			FirstName: "John",
			LastName:  ptr("Doe"),
			Username:  "johndoe",
			Email:     "john@example.com",
			Roles:     []int{1, 2},
		}

		repo.createFunc = func(_ context.Context, _ DBTX, user *User) (int, error) {
			return 10, nil
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _, _ int) ([]Role, error) {
			return []Role{{ID: 1, Name: "Admin"}}, nil
		}

		user, err := svc.Create(context.Background(), req, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.createCalled {
			t.Fatal("Create was not called")
		}
		if len(repo.createUser.Avatar) == 0 {
			t.Fatal("avatar should be non-empty when none provided")
		}
		var avatarData map[string]interface{}
		if err := json.Unmarshal(repo.createUser.Avatar, &avatarData); err != nil {
			t.Fatalf("avatar should be valid JSON: %v", err)
		}
		url, ok := avatarData["url"].(string)
		if !ok {
			t.Fatal("avatar should contain a url field")
		}
		if !strings.Contains(url, "dicebear.com") {
			t.Fatalf("avatar URL should reference dicebear.com, got: %s", url)
		}

		if !repo.attachRolesCalled {
			t.Fatal("AttachRoles was not called")
		}
		if repo.attachRolesUserID != 10 {
			t.Fatalf("expected AttachRoles userID 10, got %d", repo.attachRolesUserID)
		}
		if len(repo.attachRolesIDs) != 2 || repo.attachRolesIDs[0] != 1 || repo.attachRolesIDs[1] != 2 {
			t.Fatalf("unexpected role IDs: %v", repo.attachRolesIDs)
		}

		if user.ID != 10 {
			t.Fatalf("expected user ID 10, got %d", user.ID)
		}
		if user.FirstName != "John" {
			t.Fatalf("expected FirstName John, got %s", user.FirstName)
		}
		if user.CreatedAt.IsZero() {
			t.Fatal("CreatedAt should be set")
		}
		if user.UpdatedAt.IsZero() {
			t.Fatal("UpdatedAt should be set")
		}
		if len(user.Roles) == 0 || user.Roles[0].Name != "Admin" {
			t.Fatal("user should have Admin role attached")
		}
	})

	t.Run("uses provided avatar when supplied", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		providedAvatar := json.RawMessage(`{"url":"https://custom.example.com/avatar.jpg"}`)
		req := CreateUserRequest{
			FirstName: "Jane",
			Username:  "jane",
			Email:     "jane@example.com",
			Avatar:    providedAvatar,
			Roles:     []int{1},
		}

		repo.createFunc = func(_ context.Context, _ DBTX, user *User) (int, error) {
			return 20, nil
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _, _ int) ([]Role, error) {
			return nil, nil
		}

		user, err := svc.Create(context.Background(), req, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.createCalled {
			t.Fatal("Create was not called")
		}
		if string(repo.createUser.Avatar) != string(providedAvatar) {
			t.Fatalf("expected avatar %q, got %q", string(providedAvatar), string(repo.createUser.Avatar))
		}
		if user.ID != 20 {
			t.Fatalf("expected user ID 20, got %d", user.ID)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests — Update
// ---------------------------------------------------------------------------

func TestUpdate(t *testing.T) {
	existingUser := &User{
		ID:        5,
		TenantID:  1,
		FirstName: "Old",
		Username:  "olduser",
		Email:     "old@example.com",
		Password:  "$2a$10$existinghashforsure",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	t.Run("partial update keeps existing password", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		// GetByID is called twice: once at the start to load the existing user,
		// and once at the end via s.GetByID after commit.
		var getByIDCalls int
		repo.getByIDFunc = func(_ context.Context, _ DBTX, _, _ int) (*User, error) {
			getByIDCalls++
			if getByIDCalls == 2 {
				return &User{
					ID:        5,
					TenantID:  1,
					FirstName: "Updated",
					LastName:  ptr("User"),
					Username:  "updateduser",
					Email:     "updated@example.com",
					Password:  existingUser.Password,
					CreatedAt: time.Now().Add(-24 * time.Hour),
					UpdatedAt: time.Now(),
				}, nil
			}
			return existingUser, nil
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _, _ int) ([]Role, error) {
			return []Role{{ID: 2, Name: "Editor"}}, nil
		}

		req := UpdateUserRequest{
			FirstName: "Updated",
			LastName:  ptr("User"),
			Username:  "updateduser",
			Email:     "updated@example.com",
			Roles:     []int{2, 3},
		}

		user, err := svc.Update(context.Background(), 1, 5, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.updateCalled {
			t.Fatal("Update was not called")
		}
		if repo.updateUser.FirstName != "Updated" {
			t.Fatalf("expected FirstName Updated, got %s", repo.updateUser.FirstName)
		}
		if repo.updateUser.Password != existingUser.Password {
			t.Fatal("password should remain unchanged")
		}

		if !repo.syncRolesCalled {
			t.Fatal("SyncRoles was not called")
		}
		if repo.syncRolesUserID != 5 {
			t.Fatalf("expected SyncRoles userID 5, got %d", repo.syncRolesUserID)
		}
		if len(repo.syncRolesIDs) != 2 || repo.syncRolesIDs[0] != 2 {
			t.Fatalf("unexpected SyncRoles IDs: %v", repo.syncRolesIDs)
		}

		if user.FirstName != "Updated" {
			t.Fatalf("expected returned user FirstName Updated, got %s", user.FirstName)
		}
		if len(user.Roles) == 0 || user.Roles[0].Name != "Editor" {
			t.Fatal("returned user should have roles")
		}
	})

	t.Run("update with password change hashes new password", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		repo.getByIDFunc = func(_ context.Context, _ DBTX, _, _ int) (*User, error) {
			return existingUser, nil
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _, _ int) ([]Role, error) {
			return nil, nil
		}

		newPassword := "newpassword123"
		req := UpdateUserRequest{
			FirstName: "Updated",
			Username:  "updateduser",
			Email:     "updated@example.com",
			Password:  &newPassword,
			Roles:     []int{1},
		}

		user, err := svc.Update(context.Background(), 1, 5, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.updateCalled {
			t.Fatal("Update was not called")
		}
		if repo.updateUser.Password == newPassword {
			t.Fatal("password should be hashed, not plaintext")
		}
		if repo.updateUser.Password == existingUser.Password {
			t.Fatal("password should differ from old password")
		}
		if user == nil {
			t.Fatal("expected a user to be returned")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests — Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	t.Run("super admin id=1 returns error", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		err := svc.Delete(context.Background(), 1, 1)
		if err == nil {
			t.Fatal("expected error when deleting super admin")
		}
		if !strings.Contains(err.Error(), "super admin") {
			t.Fatalf("error should mention super admin, got: %v", err)
		}
		if repo.softDeleteCalled {
			t.Fatal("SoftDelete should not be called for super admin")
		}
	})

	t.Run("normal user is soft deleted", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		err := svc.Delete(context.Background(), 1, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !repo.softDeleteCalled {
			t.Fatal("SoftDelete was not called")
		}
		if repo.softDeleteTenantID != 1 {
			t.Fatalf("expected tenantID 1, got %d", repo.softDeleteTenantID)
		}
		if repo.softDeleteID != 5 {
			t.Fatalf("expected id 5, got %d", repo.softDeleteID)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests — List
// ---------------------------------------------------------------------------

func TestList(t *testing.T) {
	t.Run("defaults page to 1 and limit to 20 when zero", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		repo.listFunc = func(_ context.Context, _ DBTX, _ int, limit, offset int, _ string, _ *int) ([]*User, int, error) {
			return nil, 0, nil
		}

		query := ListUserQuery{Page: 0, Limit: 0}
		result, err := svc.List(context.Background(), query, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.listCalled {
			t.Fatal("List was not called")
		}
		if repo.listLimit != 20 {
			t.Fatalf("expected default limit 20, got %d", repo.listLimit)
		}
		if repo.listOffset != 0 {
			t.Fatalf("expected offset 0, got %d", repo.listOffset)
		}
		if result.Meta.CurrentPage != 1 {
			t.Fatalf("expected CurrentPage 1, got %d", result.Meta.CurrentPage)
		}
		if result.Meta.PerPage != 20 {
			t.Fatalf("expected PerPage 20, got %d", result.Meta.PerPage)
		}
		if result.Meta.From != 0 || result.Meta.To != 0 {
			t.Fatalf("expected From/To 0 for empty result, got %d/%d", result.Meta.From, result.Meta.To)
		}
	})

	t.Run("defaults limit to 20 when outside 1-100 range", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		repo.listFunc = func(_ context.Context, _ DBTX, _ int, limit, offset int, _ string, _ *int) ([]*User, int, error) {
			return nil, 0, nil
		}

		_, err := svc.List(context.Background(), ListUserQuery{Page: 1, Limit: 200}, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.listLimit != 20 {
			t.Fatalf("expected default limit 20 for out-of-range value, got %d", repo.listLimit)
		}
	})

	t.Run("batch loads roles via GetRolesByUserIDs", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		users := []*User{
			{ID: 1, FirstName: "Alice", Username: "alice", Email: "alice@test.com", TenantID: 1},
			{ID: 2, FirstName: "Bob", Username: "bob", Email: "bob@test.com", TenantID: 1},
		}
		roleMap := map[int][]Role{
			1: {{ID: 10, Name: "Admin"}},
			2: {{ID: 20, Name: "Editor"}},
		}

		repo.listFunc = func(_ context.Context, _ DBTX, _ int, _, _ int, _ string, _ *int) ([]*User, int, error) {
			return users, 2, nil
		}
		repo.getRolesByUserIDsFunc = func(_ context.Context, _ DBTX, _ int, userIDs []int) (map[int][]Role, error) {
			return roleMap, nil
		}

		result, err := svc.List(context.Background(), ListUserQuery{Page: 1, Limit: 20}, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.getRolesByUserIDsCalled {
			t.Fatal("GetRolesByUserIDs was not called")
		}
		if len(repo.getRolesByUserIDs) != 2 {
			t.Fatalf("expected 2 user IDs, got %d", len(repo.getRolesByUserIDs))
		}

		data := result.Data.([]*User)
		if len(data) != 2 {
			t.Fatalf("expected 2 users, got %d", len(data))
		}
		if len(data[0].Roles) == 0 || data[0].Roles[0].Name != "Admin" {
			t.Fatal("Alice should have Admin role")
		}
		if len(data[1].Roles) == 0 || data[1].Roles[0].Name != "Editor" {
			t.Fatal("Bob should have Editor role")
		}

		if result.Meta.Total != 2 {
			t.Fatalf("expected Total 2, got %d", result.Meta.Total)
		}
		if result.Meta.LastPage != 1 {
			t.Fatalf("expected LastPage 1, got %d", result.Meta.LastPage)
		}
		if result.Meta.From != 1 || result.Meta.To != 2 {
			t.Fatalf("expected From 1 To 2, got %d %d", result.Meta.From, result.Meta.To)
		}
	})

	t.Run("computes correct pagination for page 3 limit 10", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		users := make([]*User, 5)
		for i := range users {
			users[i] = &User{ID: i + 21}
		}

		repo.listFunc = func(_ context.Context, _ DBTX, _ int, limit, offset int, _ string, _ *int) ([]*User, int, error) {
			return users, 25, nil
		}
		repo.getRolesByUserIDsFunc = func(_ context.Context, _ DBTX, _ int, _ []int) (map[int][]Role, error) {
			return map[int][]Role{}, nil
		}

		result, err := svc.List(context.Background(), ListUserQuery{Page: 3, Limit: 10}, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if repo.listLimit != 10 {
			t.Fatalf("expected limit 10, got %d", repo.listLimit)
		}
		if repo.listOffset != 20 {
			t.Fatalf("expected offset 20 for page 3, got %d", repo.listOffset)
		}
		if result.Meta.CurrentPage != 3 {
			t.Fatalf("expected CurrentPage 3, got %d", result.Meta.CurrentPage)
		}
		if result.Meta.PerPage != 10 {
			t.Fatalf("expected PerPage 10, got %d", result.Meta.PerPage)
		}
		if result.Meta.Total != 25 {
			t.Fatalf("expected Total 25, got %d", result.Meta.Total)
		}
		if result.Meta.LastPage != 3 {
			t.Fatalf("expected LastPage 3, got %d", result.Meta.LastPage)
		}
		if result.Meta.From != 21 || result.Meta.To != 25 {
			t.Fatalf("expected From 21 To 25, got %d %d", result.Meta.From, result.Meta.To)
		}
	})
}
