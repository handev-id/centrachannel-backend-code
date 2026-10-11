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

	"github.com/google/uuid"

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

func (r *mockRows) Columns() []string              { return nil }
func (r *mockRows) Close() error                   { return nil }
func (r *mockRows) Next(dest []driver.Value) error { return io.EOF }

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
	listRolesFunc         func(context.Context, DBTX, uuid.UUID) ([]Role, error)
	listFunc              func(context.Context, DBTX, uuid.UUID, int, int, string, *uuid.UUID) ([]*User, int, error)
	getByIDFunc           func(context.Context, DBTX, uuid.UUID, uuid.UUID) (*User, error)
	createFunc            func(context.Context, DBTX, *User) (uuid.UUID, error)
	updateFunc            func(context.Context, DBTX, uuid.UUID, uuid.UUID, *User) error
	softDeleteFunc        func(context.Context, DBTX, uuid.UUID, uuid.UUID) error
	getRolesByUserIDFunc  func(context.Context, DBTX, uuid.UUID, uuid.UUID) ([]Role, error)
	getRolesByUserIDsFunc func(context.Context, DBTX, uuid.UUID, []uuid.UUID) (map[uuid.UUID][]Role, error)
	attachRolesFunc       func(context.Context, DBTX, uuid.UUID, uuid.UUID, []uuid.UUID) error
	syncRolesFunc         func(context.Context, DBTX, uuid.UUID, uuid.UUID, []uuid.UUID) error

	createCalled            bool
	createUser              *User
	attachRolesCalled       bool
	attachRolesUserID       uuid.UUID
	attachRolesIDs          []uuid.UUID
	updateCalled            bool
	updateUser              *User
	syncRolesCalled         bool
	syncRolesUserID         uuid.UUID
	syncRolesIDs            []uuid.UUID
	softDeleteCalled        bool
	softDeleteTenantID      uuid.UUID
	softDeleteID            uuid.UUID
	listCalled              bool
	listLimit               int
	listOffset              int
	getRolesByUserIDsCalled bool
	getRolesByUserIDs       []uuid.UUID
}

func (m *mockUserRepository) ListRoles(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Role, error) {
	if m.listRolesFunc != nil {
		return m.listRolesFunc(ctx, q, tenantID)
	}
	return []Role{}, nil
}

func (m *mockUserRepository) List(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, search string, roleID *uuid.UUID) ([]*User, int, error) {
	m.listCalled = true
	m.listLimit = limit
	m.listOffset = offset
	if m.listFunc != nil {
		return m.listFunc(ctx, q, tenantID, limit, offset, search, roleID)
	}
	return nil, 0, nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, tenantID, id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepository) Create(ctx context.Context, q DBTX, user *User) (uuid.UUID, error) {
	m.createCalled = true
	m.createUser = user
	if m.createFunc != nil {
		return m.createFunc(ctx, q, user)
	}
	return uuid.Nil, nil
}

func (m *mockUserRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, user *User) error {
	m.updateCalled = true
	m.updateUser = user
	if m.updateFunc != nil {
		return m.updateFunc(ctx, q, tenantID, id, user)
	}
	return nil
}

func (m *mockUserRepository) SoftDelete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	m.softDeleteCalled = true
	m.softDeleteTenantID = tenantID
	m.softDeleteID = id
	if m.softDeleteFunc != nil {
		return m.softDeleteFunc(ctx, q, tenantID, id)
	}
	return nil
}

func (m *mockUserRepository) GetRolesByUserID(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID) ([]Role, error) {
	if m.getRolesByUserIDFunc != nil {
		return m.getRolesByUserIDFunc(ctx, q, tenantID, userID)
	}
	return nil, nil
}

func (m *mockUserRepository) GetRolesByUserIDs(ctx context.Context, q DBTX, tenantID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID][]Role, error) {
	m.getRolesByUserIDsCalled = true
	m.getRolesByUserIDs = userIDs
	if m.getRolesByUserIDsFunc != nil {
		return m.getRolesByUserIDsFunc(ctx, q, tenantID, userIDs)
	}
	return map[uuid.UUID][]Role{}, nil
}

func (m *mockUserRepository) AttachRoles(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID, roleIDs []uuid.UUID) error {
	m.attachRolesCalled = true
	m.attachRolesUserID = userID
	m.attachRolesIDs = roleIDs
	if m.attachRolesFunc != nil {
		return m.attachRolesFunc(ctx, q, tenantID, userID, roleIDs)
	}
	return nil
}

func (m *mockUserRepository) SyncRoles(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID, roleIDs []uuid.UUID) error {
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

var testTenant = &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Test Tenant"}

// ---------------------------------------------------------------------------
// Tests — Create
// ---------------------------------------------------------------------------

func TestCreate(t *testing.T) {
	t.Run("creates user without avatar when none provided", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		req := CreateUserRequest{
			FirstName: "John",
			LastName:  ptr("Doe"),
			Username:  "johndoe",
			Email:     "john@example.com",
			Password:  "password123",
			Roles:     []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000002")},
		}

		repo.createFunc = func(_ context.Context, _ DBTX, user *User) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-00000000000a"), nil
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID) ([]Role, error) {
			return []Role{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Admin"}}, nil
		}

		user, err := svc.Create(context.Background(), req, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.createCalled {
			t.Fatal("Create was not called")
		}
		if len(repo.createUser.Avatar) != 0 {
			t.Fatalf("expected empty avatar when none provided, got %s", string(repo.createUser.Avatar))
		}

		if !repo.attachRolesCalled {
			t.Fatal("AttachRoles was not called")
		}
		if repo.attachRolesUserID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
			t.Fatalf("expected AttachRoles userID 10, got %s", repo.attachRolesUserID)
		}
		if len(repo.attachRolesIDs) != 2 || repo.attachRolesIDs[0] != uuid.MustParse("00000000-0000-4000-8000-000000000001") || repo.attachRolesIDs[1] != uuid.MustParse("00000000-0000-4000-8000-000000000002") {
			t.Fatalf("unexpected role IDs: %v", repo.attachRolesIDs)
		}

		if user.ID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
			t.Fatalf("expected user ID 10, got %s", user.ID)
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
			Password:  "password456",
			Avatar:    providedAvatar,
			Roles:     []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000001")},
		}

		repo.createFunc = func(_ context.Context, _ DBTX, user *User) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-000000000014"), nil
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID) ([]Role, error) {
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
		if user.ID != uuid.MustParse("00000000-0000-4000-8000-000000000014") {
			t.Fatalf("expected user ID 20, got %s", user.ID)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests — Update
// ---------------------------------------------------------------------------

func TestUpdate(t *testing.T) {
	existingUser := &User{
		ID:        uuid.MustParse("00000000-0000-4000-8000-000000000005"),
		TenantID:  uuid.MustParse("00000000-0000-4000-8000-000000000001"),
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
		repo.getByIDFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID) (*User, error) {
			getByIDCalls++
			if getByIDCalls == 2 {
				return &User{
					ID:        uuid.MustParse("00000000-0000-4000-8000-000000000005"),
					TenantID:  uuid.MustParse("00000000-0000-4000-8000-000000000001"),
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
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID) ([]Role, error) {
			return []Role{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), Name: "Editor"}}, nil
		}

		req := UpdateUserRequest{
			FirstName: "Updated",
			LastName:  ptr("User"),
			Username:  "updateduser",
			Email:     "updated@example.com",
			Roles:     []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000003")},
		}

		user, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), req)
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
		if repo.syncRolesUserID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Fatalf("expected SyncRoles userID 5, got %s", repo.syncRolesUserID)
		}
		if len(repo.syncRolesIDs) != 2 || repo.syncRolesIDs[0] != uuid.MustParse("00000000-0000-4000-8000-000000000002") {
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

		repo.getByIDFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID) (*User, error) {
			return existingUser, nil
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID) ([]Role, error) {
			return nil, nil
		}

		newPassword := "newpassword123"
		req := UpdateUserRequest{
			FirstName: "Updated",
			Username:  "updateduser",
			Email:     "updated@example.com",
			Password:  &newPassword,
			Roles:     []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000001")},
		}

		user, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), req)
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
	t.Run("super admin id=00000000-0000-4000-8000-000000000001 returns error", func(t *testing.T) {
		repo := &mockUserRepository{
			getByIDFunc: func(_ context.Context, _ DBTX, tenantID, id uuid.UUID) (*User, error) {
				return &User{ID: id, TenantID: tenantID}, nil
			},
		}
		repo.getRolesByUserIDFunc = func(_ context.Context, _ DBTX, _, _ uuid.UUID) ([]Role, error) {
			return []Role{{Name: "super-admin"}}, nil
		}
		svc := newTestService(repo)

		err := svc.Delete(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
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
		repo := &mockUserRepository{
			getByIDFunc: func(_ context.Context, _ DBTX, tenantID, id uuid.UUID) (*User, error) {
				return &User{ID: id, TenantID: tenantID}, nil
			},
		}
		svc := newTestService(repo)

		err := svc.Delete(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !repo.softDeleteCalled {
			t.Fatal("SoftDelete was not called")
		}
		if repo.softDeleteTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Fatalf("expected tenantID 1, got %s", repo.softDeleteTenantID)
		}
		if repo.softDeleteID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Fatalf("expected id 5, got %s", repo.softDeleteID)
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

		repo.listFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, limit int, offset int, _ string, _ *uuid.UUID) ([]*User, int, error) {
			return nil, 0, nil
		}

		query := ListUserQuery{Page: 0, Limit: 0}
		result, total, err := svc.List(context.Background(), query, testTenant)
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
		if total != 0 {
			t.Fatalf("expected total 0, got %d", total)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 users, got %d", len(result))
		}
	})

	t.Run("defaults limit to 20 when outside 1-100 range", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		repo.listFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, limit int, offset int, _ string, _ *uuid.UUID) ([]*User, int, error) {
			return nil, 0, nil
		}

		_, _, err := svc.List(context.Background(), ListUserQuery{Page: 1, Limit: 200}, testTenant)
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
			{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "Alice", Username: "alice", Email: "alice@test.com", TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001")},
			{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), FirstName: "Bob", Username: "bob", Email: "bob@test.com", TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001")},
		}
		roleMap := map[uuid.UUID][]Role{uuid.MustParse("00000000-0000-4000-8000-000000000001"): {{ID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), Name: "Admin"}}, uuid.MustParse("00000000-0000-4000-8000-000000000002"): {{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), Name: "Editor"}}}

		repo.listFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ int, _ int, _ string, _ *uuid.UUID) ([]*User, int, error) {
			return users, 2, nil
		}
		repo.getRolesByUserIDsFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID][]Role, error) {
			return roleMap, nil
		}
		result, total, err := svc.List(context.Background(), ListUserQuery{Page: 1, Limit: 20}, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !repo.getRolesByUserIDsCalled {
			t.Fatal("GetRolesByUserIDs was not called")
		}
		if len(repo.getRolesByUserIDs) != 2 {
			t.Fatalf("expected 2 user IDs, got %d", len(repo.getRolesByUserIDs))

		}

		if len(result) != 2 {
			t.Fatalf("expected 2 users, got %d", len(result))
		}
		if len(result[0].Roles) == 0 || result[0].Roles[0].Name != "Admin" {
			t.Fatal("Alice should have Admin role")
		}
		if len(result[1].Roles) == 0 || result[1].Roles[0].Name != "Editor" {
			t.Fatal("Bob should have Editor role")
		}

		if total != 2 {
			t.Fatalf("expected total 2, got %d", total)
		}
	})

	t.Run("computes correct pagination for page 3 limit 10", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := newTestService(repo)

		users := make([]*User, 5)
		for i := range users {
			users[i] = &User{ID: uuid.New()}
		}

		repo.listFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, limit int, offset int, _ string, _ *uuid.UUID) ([]*User, int, error) {
			return users, 25, nil
		}
		repo.getRolesByUserIDsFunc = func(_ context.Context, _ DBTX, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID][]Role, error) {
			return map[uuid.UUID][]Role{}, nil
		}

		result, total, err := svc.List(context.Background(), ListUserQuery{Page: 3, Limit: 10}, testTenant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if repo.listLimit != 10 {
			t.Fatalf("expected limit 10, got %d", repo.listLimit)
		}
		if repo.listOffset != 20 {
			t.Fatalf("expected offset 20 for page 3, got %d", repo.listOffset)
		}
		if total != 25 {
			t.Fatalf("expected total 25, got %d", total)
		}
		if len(result) != 5 {
			t.Fatalf("expected 5 users, got %d", len(result))
		}
	})
}

func TestListRoles(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	expected := []Role{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000003"), TenantID: tenantID, Name: "agent"}}
	repo := &mockUserRepository{listRolesFunc: func(_ context.Context, _ DBTX, gotTenantID uuid.UUID) ([]Role, error) {
		if gotTenantID != tenantID {
			t.Fatalf("expected tenant %s, got %s", tenantID, gotTenantID)
		}
		return expected, nil
	}}
	svc := newTestService(repo)
	defer svc.db.Close()
	roles, err := svc.ListRoles(context.Background(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0].ID != expected[0].ID {
		t.Fatalf("unexpected roles: %+v", roles)
	}
}
