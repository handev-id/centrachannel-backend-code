package contact

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"centrachannel/config"
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
	_ driver.Conn          = (*mockConn)(nil)
	_ driver.ConnBeginTx   = (*mockConn)(nil)
	_ driver.ExecerContext = (*mockConn)(nil)
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

// ---- Repository Mock ----

type mockContactRepository struct {
	listFunc        func(ctx context.Context, q DBTX, tenantID int, limit, offset int, search, status string, channelID int) ([]*Contact, int, error)
	getByIDFunc     func(ctx context.Context, q DBTX, tenantID int, id int) (*Contact, error)
	createFunc      func(ctx context.Context, q DBTX, contact *Contact) (int, error)
	updateFunc      func(ctx context.Context, q DBTX, tenantID int, id int, contact *Contact) error
	softDeleteFunc  func(ctx context.Context, q DBTX, tenantID int, id int) error
	getByPhoneFunc  func(ctx context.Context, q DBTX, tenantID int, phone string) (*Contact, error)
}

func (m *mockContactRepository) List(ctx context.Context, q DBTX, tenantID int, limit, offset int, search, status string, channelID int) ([]*Contact, int, error) {
	return m.listFunc(ctx, q, tenantID, limit, offset, search, status, channelID)
}

func (m *mockContactRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Contact, error) {
	return m.getByIDFunc(ctx, q, tenantID, id)
}

func (m *mockContactRepository) Create(ctx context.Context, q DBTX, contact *Contact) (int, error) {
	return m.createFunc(ctx, q, contact)
}

func (m *mockContactRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, contact *Contact) error {
	return m.updateFunc(ctx, q, tenantID, id, contact)
}

func (m *mockContactRepository) SoftDelete(ctx context.Context, q DBTX, tenantID int, id int) error {
	return m.softDeleteFunc(ctx, q, tenantID, id)
}

func (m *mockContactRepository) GetByPhone(ctx context.Context, q DBTX, tenantID int, phone string) (*Contact, error) {
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
	return NewContactService(repo, db, &config.Config{}, logger.NewLogger("error", "json")).(*contactService)
}

func strPtr(s string) *string { return &s }

// ---- Tests ----

func TestContactService_Create(t *testing.T) {
	t.Run("success with dicebear avatar and default status", func(t *testing.T) {
		var captured *Contact
		repo := &mockContactRepository{
			createFunc: func(ctx context.Context, q DBTX, contact *Contact) (int, error) {
				captured = contact
				return 1, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		req := CreateContactRequest{FirstName: "John"}
		ten := &tenant.Tenant{ID: 1}

		result, err := svc.Create(context.Background(), req, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "individual" {
			t.Errorf("expected status 'individual', got %q", result.Status)
		}
		var av map[string]string
		if err := json.Unmarshal(captured.Avatar, &av); err != nil {
			t.Fatalf("failed to unmarshal avatar: %v", err)
		}
		if !strings.Contains(av["url"], "dicebear.com") {
			t.Errorf("expected DiceBear avatar URL, got %s", av["url"])
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
			ID: 1, TenantID: 1, FirstName: "John",
			LastName: &lastName, Username: strPtr("johndoe"),
			Status: "individual",
		}
		var updated *Contact
		repo := &mockContactRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Contact, error) {
				return existing, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID int, id int, contact *Contact) error {
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

		result, err := svc.Update(context.Background(), 1, 1, req)
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
		var capturedTenantID, capturedID int
		repo := &mockContactRepository{
			softDeleteFunc: func(ctx context.Context, q DBTX, tenantID int, id int) error {
				capturedTenantID = tenantID
				capturedID = id
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		if err := svc.Delete(context.Background(), 1, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if capturedID != 5 {
			t.Errorf("expected id 5, got %d", capturedID)
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

		err := svc.Merge(context.Background(), 1, 5, 5)
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
		}
		db := newMockDB(m)
		defer db.Close()
		repo := &mockContactRepository{}
		svc := newService(repo, db)

		if err := svc.Merge(context.Background(), 1, 5, 10); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if execCalls != 3 {
			t.Errorf("expected 3 exec calls, got %d", execCalls)
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

		if err := svc.Unmerge(context.Background(), 1, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if execCalls != 2 {
			t.Errorf("expected 2 exec calls, got %d", execCalls)
		}
	})
}

func TestContactService_List(t *testing.T) {
	t.Run("pagination defaults", func(t *testing.T) {
		contacts := []*Contact{{ID: 1, FirstName: "John"}}
		var capturedLimit, capturedOffset int
		repo := &mockContactRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit, offset int, search, status string, channelID int) ([]*Contact, int, error) {
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
		ten := &tenant.Tenant{ID: 1}

		result, err := svc.List(context.Background(), q, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Meta.CurrentPage != 1 {
			t.Errorf("expected CurrentPage 1, got %d", result.Meta.CurrentPage)
		}
		if result.Meta.PerPage != 20 {
			t.Errorf("expected PerPage 20, got %d", result.Meta.PerPage)
		}
		if capturedLimit != 20 {
			t.Errorf("expected limit 20, got %d", capturedLimit)
		}
		if capturedOffset != 0 {
			t.Errorf("expected offset 0, got %d", capturedOffset)
		}
	})

	t.Run("search status channel filters", func(t *testing.T) {
		var capturedSearch, capturedStatus string
		var capturedChannelID int
		repo := &mockContactRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit, offset int, search, status string, channelID int) ([]*Contact, int, error) {
				capturedSearch = search
				capturedStatus = status
				capturedChannelID = channelID
				return []*Contact{}, 0, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		q := ListContactQuery{Page: 1, Limit: 10, Search: "john", Status: "active", ChannelID: 3}
		ten := &tenant.Tenant{ID: 1}

		_, err := svc.List(context.Background(), q, ten)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedSearch != "john" {
			t.Errorf("expected search 'john', got %q", capturedSearch)
		}
		if capturedStatus != "active" {
			t.Errorf("expected status 'active', got %q", capturedStatus)
		}
		if capturedChannelID != 3 {
			t.Errorf("expected channelID 3, got %d", capturedChannelID)
		}
	})
}

func TestContactService_GetConversations(t *testing.T) {
	t.Run("returns conversation list via raw SQL query", func(t *testing.T) {
		now := time.Now()
		m := &mockConn{
			queryFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				return &mockRows{
					columns: []string{"id", "status", "profile_id", "agent_id", "channel_id", "unread_count", "last_message", "last_activity", "created_at"},
					data: [][]driver.Value{
						{int64(1), "active", int64(10), int64(42), int64(5), int64(3), `{"text":"hello"}`, now, now},
						{int64(2), "closed", int64(11), nil, int64(6), int64(0), nil, nil, now},
					},
				}, nil
			},
		}
		db := newMockDB(m)
		defer db.Close()
		repo := &mockContactRepository{}
		svc := newService(repo, db)

		result, err := svc.GetConversations(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rv := reflect.ValueOf(result)
		if rv.Kind() != reflect.Slice {
			t.Fatalf("expected slice, got %T", result)
		}
		if rv.Len() != 2 {
			t.Fatalf("expected 2 conversations, got %d", rv.Len())
		}

		// First conversation — has agent_id, last_message, last_activity
		c0 := rv.Index(0)
		if c0.FieldByName("ID").Int() != 1 {
			t.Errorf("expected ID 1, got %d", c0.FieldByName("ID").Int())
		}
		if c0.FieldByName("Status").String() != "active" {
			t.Errorf("expected Status 'active', got %q", c0.FieldByName("Status").String())
		}
		if c0.FieldByName("UnreadCount").Int() != 3 {
			t.Errorf("expected UnreadCount 3, got %d", c0.FieldByName("UnreadCount").Int())
		}
		agentID := c0.FieldByName("AgentID")
		if agentID.IsNil() {
			t.Error("expected AgentID to be non-nil for first conversation")
		} else if agentID.Elem().Int() != 42 {
			t.Errorf("expected AgentID 42, got %d", agentID.Elem().Int())
		}
		lastMsg := c0.FieldByName("LastMessage")
		if lastMsg.IsNil() {
			t.Error("expected LastMessage to be non-nil for first conversation")
		}
		lastAct := c0.FieldByName("LastActivity")
		if lastAct.IsNil() {
			t.Error("expected LastActivity to be non-nil for first conversation")
		}

		// Second conversation — no agent_id, no last_message, no last_activity
		c1 := rv.Index(1)
		if c1.FieldByName("ID").Int() != 2 {
			t.Errorf("expected ID 2, got %d", c1.FieldByName("ID").Int())
		}
		if c1.FieldByName("Status").String() != "closed" {
			t.Errorf("expected Status 'closed', got %q", c1.FieldByName("Status").String())
		}
		if !c1.FieldByName("AgentID").IsNil() {
			t.Error("expected AgentID to be nil for second conversation")
		}
		if !c1.FieldByName("LastMessage").IsNil() {
			t.Error("expected LastMessage to be nil for second conversation")
		}
		if !c1.FieldByName("LastActivity").IsNil() {
			t.Error("expected LastActivity to be nil for second conversation")
		}
	})
}
