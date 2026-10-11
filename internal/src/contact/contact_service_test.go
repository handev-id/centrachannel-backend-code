package contact

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

// ---- Driver-level mocks ----

var mockDriverInst = &mockDriver{}

func init() {
	for _, name := range sql.Drivers() {
		if name == "mock" {
			return
		}
	}
	sql.Register("mock", mockDriverInst)
}

type mockDriver struct {
	conn *mockConn
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return d.conn, nil
}

type mockConn struct {
	beginTxFunc func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)
	execFunc    func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error)
	queryFunc   func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error)
	prepareFunc func(query string) (driver.Stmt, error)
	closeFunc   func() error
	beginFunc   func() (driver.Tx, error)
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	if c.prepareFunc != nil {
		return c.prepareFunc(query)
	}
	return nil, fmt.Errorf("Prepare not implemented")
}

func (c *mockConn) Close() error {
	if c.closeFunc != nil {
		return c.closeFunc()
	}
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	if c.beginFunc != nil {
		return c.beginFunc()
	}
	return nil, fmt.Errorf("Begin not implemented")
}

func (c *mockConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.beginTxFunc != nil {
		return c.beginTxFunc(ctx, opts)
	}
	return nil, fmt.Errorf("BeginTx not implemented")
}

func (c *mockConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.execFunc != nil {
		return c.execFunc(ctx, query, args)
	}
	return nil, fmt.Errorf("ExecContext not implemented")
}

func (c *mockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.queryFunc != nil {
		return c.queryFunc(ctx, query, args)
	}
	return nil, fmt.Errorf("QueryContext not implemented")
}

var (
	_ driver.Conn           = (*mockConn)(nil)
	_ driver.ConnBeginTx    = (*mockConn)(nil)
	_ driver.ExecerContext  = (*mockConn)(nil)
	_ driver.QueryerContext = (*mockConn)(nil)
)

type mockTx struct {
	commitFunc   func() error
	rollbackFunc func() error
}

func (t *mockTx) Commit() error {
	if t.commitFunc != nil {
		return t.commitFunc()
	}
	return nil
}

func (t *mockTx) Rollback() error {
	if t.rollbackFunc != nil {
		return t.rollbackFunc()
	}
	return nil
}

var _ driver.Tx = (*mockTx)(nil)

type mockRows struct {
	columns []string
	data    [][]driver.Value
	pos     int
}

func (r *mockRows) Columns() []string { return r.columns }

func (r *mockRows) Close() error { return nil }

func (r *mockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

var _ driver.Rows = (*mockRows)(nil)

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r *mockResult) LastInsertId() (int64, error) { return r.lastInsertID, nil }

func (r *mockResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

var _ driver.Result = (*mockResult)(nil)

// ---- Profile Repository Mock ----

type mockProfileRepo struct {
	getByContactIDFunc  func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID uuid.UUID) ([]profile.Profile, error)
	getByContactIDsFunc func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID][]profile.Profile, error)
}

func (m *mockProfileRepo) List(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID, channelID uuid.UUID) ([]profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID) (*profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByExternalIDAndChannelID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) Create(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockProfileRepo) Update(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID, p *profile.Profile) error {
	return nil
}
func (m *mockProfileRepo) GetByContactID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID uuid.UUID) ([]profile.Profile, error) {
	if m.getByContactIDFunc != nil {
		return m.getByContactIDFunc(ctx, q, tenantID, contactID)
	}
	return []profile.Profile{}, nil
}
func (m *mockProfileRepo) GetByContactIDs(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID][]profile.Profile, error) {
	if m.getByContactIDsFunc != nil {
		return m.getByContactIDsFunc(ctx, q, tenantID, contactIDs)
	}
	return map[uuid.UUID][]profile.Profile{}, nil
}

// ---- Repository Mock ----

type mockContactRepository struct {
	listFunc       func(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, f ListContactQuery) ([]*Contact, int, error)
	getByIDFunc    func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Contact, error)
	createFunc     func(ctx context.Context, q DBTX, contact *Contact) (uuid.UUID, error)
	updateFunc     func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, contact *Contact) error
	softDeleteFunc func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	getByPhoneFunc func(ctx context.Context, q DBTX, tenantID uuid.UUID, phone string) (*Contact, error)
}

func (m *mockContactRepository) List(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, f ListContactQuery) ([]*Contact, int, error) {
	return m.listFunc(ctx, q, tenantID, limit, offset, f)
}

func (m *mockContactRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Contact, error) {
	return m.getByIDFunc(ctx, q, tenantID, id)
}

func (m *mockContactRepository) Create(ctx context.Context, q DBTX, contact *Contact) (uuid.UUID, error) {
	return m.createFunc(ctx, q, contact)
}

func (m *mockContactRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, contact *Contact) error {
	return m.updateFunc(ctx, q, tenantID, id, contact)
}

func (m *mockContactRepository) SoftDelete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return m.softDeleteFunc(ctx, q, tenantID, id)
}

func (m *mockContactRepository) GetByPhone(ctx context.Context, q DBTX, tenantID uuid.UUID, phone string) (*Contact, error) {
	return m.getByPhoneFunc(ctx, q, tenantID, phone)
}

// ---- Test Helpers ----

func newMockDB(conn *mockConn) *sql.DB {
	mockDriverInst.conn = conn
	db, err := sql.Open("mock", "")
	if err != nil {
		panic(fmt.Sprintf("failed to open mock db: %v", err))
	}
	db.SetMaxOpenConns(1)
	return db
}

func newService(repo ContactRepository, db *sql.DB) *contactService {
	return NewContactService(repo, &mockProfileRepo{}, db, &config.Config{}, logger.NewLogger("error", "json")).(*contactService)
}

func strPtr(s string) *string { return &s }

// ---- Tests ----

func TestContactService_Create(t *testing.T) {
	t.Run("success with default status", func(t *testing.T) {
		var captured *Contact
		repo := &mockContactRepository{
			createFunc: func(ctx context.Context, q DBTX, contact *Contact) (uuid.UUID, error) {
				captured = contact
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		req := CreateContactRequest{FirstName: "John"}
		ten := &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001")}

		result, err := svc.Create(context.Background(), req, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "individual" {
			t.Errorf("expected status 'individual', got %q", result.Status)
		}
		if captured.Avatar != nil {
			t.Errorf("expected nil avatar when none provided, got %s", string(captured.Avatar))
		}
		if captured.FirstName != "John" {
			t.Errorf("expected FirstName 'John', got %q", captured.FirstName)
		}
		if captured.ID != result.ID {
			t.Errorf("expected returned ID to match captured ID")
		}
	})
}

func TestContactService_Update(t *testing.T) {
	t.Run("partial field updates", func(t *testing.T) {
		lastName := "Doe"
		existing := &Contact{
			ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "John",
			LastName: &lastName, Username: strPtr("johndoe"),
			Status: "individual",
		}
		var updated *Contact
		repo := &mockContactRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Contact, error) {
				return existing, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, contact *Contact) error {
				updated = contact
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		newFirstName := "Jane"
		emptyLastName := ""
		req := UpdateContactRequest{
			FirstName: &newFirstName,
			LastName:  &emptyLastName,
		}

		result, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.FirstName != "Jane" {
			t.Errorf("expected FirstName 'Jane', got %q", result.FirstName)
		}
		if result.LastName == nil || *result.LastName != "" {
			t.Errorf("expected LastName to be empty string, got %v", result.LastName)
		}
		if *result.Username != "johndoe" {
			t.Errorf("expected Username unchanged 'johndoe', got %q", *result.Username)
		}
		if updated != nil {
			if updated.FirstName != "Jane" {
				t.Errorf("expected captured contact FirstName 'Jane', got %q", updated.FirstName)
			}
			if *updated.LastName != "" {
				t.Errorf("expected captured contact LastName to be empty, got %q", *updated.LastName)
			}
			if *updated.Username != "johndoe" {
				t.Errorf("expected captured Username unchanged 'johndoe', got %q", *updated.Username)
			}
		}
	})
}

func TestContactService_Delete(t *testing.T) {
	t.Run("soft delete delegation", func(t *testing.T) {
		var capturedTenantID, capturedID uuid.UUID
		repo := &mockContactRepository{
			softDeleteFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
				capturedTenantID = tenantID
				capturedID = id
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		if err := svc.Delete(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected tenantID 1, got %s", capturedTenantID)
		}
		if capturedID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Errorf("expected id 5, got %s", capturedID)
		}
	})
}

func TestContactService_Merge(t *testing.T) {
	t.Run("self merge returns error", func(t *testing.T) {
		repo := &mockContactRepository{}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		err := svc.Merge(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), uuid.MustParse("00000000-0000-4000-8000-000000000005"))
		if err == nil {
			t.Fatal("expected error for self-merge")
		}
		if err.Error() != "cannot merge contact into itself" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("success with profiles reassignment", func(t *testing.T) {
		execCalls := 0
		m := &mockConn{
			beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
				return &mockTx{
					commitFunc: func() error { return nil },
				}, nil
			},
			execFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
				execCalls++
				return &mockResult{rowsAffected: 1}, nil
			},
			queryFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				if strings.Contains(query, "SELECT EXISTS") {
					return &mockRows{columns: []string{"exists"}, data: [][]driver.Value{{false}}}, nil
				}
				return &mockRows{columns: []string{"id"}, data: [][]driver.Value{{args[0].Value}, {args[1].Value}}}, nil
			},
		}
		db := newMockDB(m)
		defer db.Close()
		repo := &mockContactRepository{}
		svc := newService(repo, db)

		if err := svc.Merge(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), uuid.MustParse("00000000-0000-4000-8000-00000000000a")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if execCalls != 2 {
			t.Errorf("expected 2 exec calls, got %d", execCalls)
		}
	})
}

func TestContactService_Unmerge(t *testing.T) {
	t.Run("clears merged_to_id and profiles", func(t *testing.T) {
		execCalls := 0
		m := &mockConn{
			beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
				return &mockTx{
					commitFunc: func() error { return nil },
				}, nil
			},
			execFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
				execCalls++
				return &mockResult{rowsAffected: 1}, nil
			},
		}
		db := newMockDB(m)
		defer db.Close()
		repo := &mockContactRepository{}
		svc := newService(repo, db)

		if err := svc.Unmerge(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if execCalls != 2 {
			t.Errorf("expected 2 exec calls, got %d", execCalls)
		}
	})
}

func TestContactService_List(t *testing.T) {
	t.Run("pagination defaults", func(t *testing.T) {
		contacts := []*Contact{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "John"}}
		var capturedLimit, capturedOffset int
		repo := &mockContactRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, f ListContactQuery) ([]*Contact, int, error) {
				capturedLimit = limit
				capturedOffset = offset
				return contacts, 1, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		q := ListContactQuery{Page: 0, Limit: 0}
		ten := &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001")}

		result, total, err := svc.List(context.Background(), q, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 contact, got %d", len(result))
		}
		if capturedLimit != 20 {
			t.Errorf("expected limit 20, got %d", capturedLimit)
		}
		if capturedOffset != 0 {
			t.Errorf("expected offset 0, got %d", capturedOffset)
		}
	})

	t.Run("search status channel filters", func(t *testing.T) {
		var capturedF ListContactQuery
		repo := &mockContactRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, f ListContactQuery) ([]*Contact, int, error) {
				capturedF = f
				return []*Contact{}, 0, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		q := ListContactQuery{Page: 1, Limit: 10, Search: "john", Status: "active", ChannelID: uuid.MustParse("00000000-0000-4000-8000-000000000003")}
		ten := &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001")}

		_, _, err := svc.List(context.Background(), q, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedF.Search != "john" {
			t.Errorf("expected search 'john', got %q", capturedF.Search)
		}
		if capturedF.Status != "active" {
			t.Errorf("expected status 'active', got %q", capturedF.Status)
		}
		if capturedF.ChannelID != uuid.MustParse("00000000-0000-4000-8000-000000000003") {
			t.Errorf("expected channelID 3, got %s", capturedF.ChannelID)
		}
	})
}

func TestContactService_GetConversations(t *testing.T) {
	t.Run("returns paginated conversation list", func(t *testing.T) {
		now := time.Now()
		queryCount := 0
		m := &mockConn{
			queryFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				queryCount++
				// First call is COUNT, second call is data
				if queryCount == 1 {
					return &mockRows{
						columns: []string{"count"},
						data:    [][]driver.Value{{int64(2)}},
					}, nil
				}
				return &mockRows{
					columns: []string{"id", "status", "profile_id", "agent_id", "channel_id", "unread_count", "last_message", "last_activity", "created_at"},
					data: [][]driver.Value{
						{"00000000-0000-4000-8000-000000000001", "active", "00000000-0000-4000-8000-00000000000a", "00000000-0000-4000-8000-00000000002a", "00000000-0000-4000-8000-000000000005", int64(3), `{"text":"hello"}`, now, now},
						{"00000000-0000-4000-8000-000000000002", "closed", "00000000-0000-4000-8000-00000000000b", nil, "00000000-0000-4000-8000-000000000006", int64(0), nil, nil, now},
					},
				}, nil
			},
		}
		db := newMockDB(m)
		defer db.Close()
		repo := &mockContactRepository{}
		svc := newService(repo, db)

		convs, total, err := svc.GetConversations(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
		if len(convs) != 2 {
			t.Fatalf("expected 2 conversations, got %d", len(convs))
		}

		// First conversation — has agent_id, last_message, last_activity
		c0 := convs[0]
		if c0.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected ID 1, got %s", c0.ID)
		}
		if c0.Status != "active" {
			t.Errorf("expected Status 'active', got %q", c0.Status)
		}
		if c0.UnreadCount != 3 {
			t.Errorf("expected UnreadCount 3, got %d", c0.UnreadCount)
		}
		if c0.AgentID == nil {
			t.Error("expected AgentID to be non-nil for first conversation")
		} else if *c0.AgentID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
			t.Errorf("expected AgentID 42, got %s", *c0.AgentID)
		}

		// Second conversation — no agent_id, no last_message, no last_activity
		c1 := convs[1]
		if c1.ID != uuid.MustParse("00000000-0000-4000-8000-000000000002") {
			t.Errorf("expected ID 2, got %s", c1.ID)
		}
		if c1.Status != "closed" {
			t.Errorf("expected Status 'closed', got %q", c1.Status)
		}
		if c1.AgentID != nil {
			t.Error("expected AgentID to be nil for second conversation")
		}
	})
}

func (m *mockContactRepository) ListConversations(ctx context.Context, q DBTX, tenantID, contactID uuid.UUID, limit, offset int) ([]ConversationBrief, int, error) {
	return NewContactRepository().ListConversations(ctx, q, tenantID, contactID, limit, offset)
}
func (m *mockContactRepository) Merge(ctx context.Context, db *sql.DB, tenantID, sourceID, targetID uuid.UUID) error {
	return NewContactRepository().Merge(ctx, db, tenantID, sourceID, targetID)
}
func (m *mockContactRepository) Unmerge(ctx context.Context, db *sql.DB, tenantID, id uuid.UUID) error {
	return NewContactRepository().Unmerge(ctx, db, tenantID, id)
}
