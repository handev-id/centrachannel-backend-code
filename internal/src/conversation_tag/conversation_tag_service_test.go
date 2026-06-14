package conversation_tag

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"testing"
)

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
func (r *mockRows) Close() error      { return nil }
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

type mockConversationTagRepository struct {
	listByConversationFunc func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]ConversationTag, error)
	attachFunc             func(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error
	detachFunc             func(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error
}

func (m *mockConversationTagRepository) ListByConversation(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]ConversationTag, error) {
	return m.listByConversationFunc(ctx, q, tenantID, conversationID)
}

func (m *mockConversationTagRepository) Attach(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error {
	return m.attachFunc(ctx, q, tenantID, conversationID, tagID)
}

func (m *mockConversationTagRepository) Detach(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error {
	return m.detachFunc(ctx, q, tenantID, conversationID, tagID)
}

func newMockDB(conn *mockConn) *sql.DB {
	mockDriverInst.conn = conn
	db, err := sql.Open("mock", "")
	if err != nil {
		panic(fmt.Sprintf("failed to open mock db: %v", err))
	}
	db.SetMaxOpenConns(1)
	return db
}

func newService(repo ConversationTagRepository, db *sql.DB) ConversationTagService {
	return NewConversationTagService(repo, db)
}

func TestConversationTagService_Attach(t *testing.T) {
	t.Run("creates tag-conversation link", func(t *testing.T) {
		var capturedTenantID, capturedConversationID, capturedTagID int
		repo := &mockConversationTagRepository{
			attachFunc: func(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error {
				capturedTenantID = tenantID
				capturedConversationID = conversationID
				capturedTagID = tagID
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		err := svc.Attach(context.Background(), 1, 10, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if capturedConversationID != 10 {
			t.Errorf("expected conversationID 10, got %d", capturedConversationID)
		}
		if capturedTagID != 5 {
			t.Errorf("expected tagID 5, got %d", capturedTagID)
		}
	})
}

func TestConversationTagService_Detach(t *testing.T) {
	t.Run("removes tag-conversation link", func(t *testing.T) {
		var capturedTenantID, capturedConversationID, capturedTagID int
		repo := &mockConversationTagRepository{
			detachFunc: func(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error {
				capturedTenantID = tenantID
				capturedConversationID = conversationID
				capturedTagID = tagID
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		err := svc.Detach(context.Background(), 1, 10, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if capturedConversationID != 10 {
			t.Errorf("expected conversationID 10, got %d", capturedConversationID)
		}
		if capturedTagID != 5 {
			t.Errorf("expected tagID 5, got %d", capturedTagID)
		}
	})
}

func TestConversationTagService_ListByConversation(t *testing.T) {
	t.Run("returns tags for a conversation", func(t *testing.T) {
		tags := []ConversationTag{
			{ID: 1, TagID: 5, ConversationID: 10, TenantID: 1},
			{ID: 2, TagID: 8, ConversationID: 10, TenantID: 1},
		}
		var capturedTenantID, capturedConversationID int
		repo := &mockConversationTagRepository{
			listByConversationFunc: func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]ConversationTag, error) {
				capturedTenantID = tenantID
				capturedConversationID = conversationID
				return tags, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		result, err := svc.ListByConversation(context.Background(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if capturedConversationID != 10 {
			t.Errorf("expected conversationID 10, got %d", capturedConversationID)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(result))
		}
		if result[0].TagID != 5 {
			t.Errorf("expected TagID 5, got %d", result[0].TagID)
		}
		if result[1].TagID != 8 {
			t.Errorf("expected TagID 8, got %d", result[1].TagID)
		}
	})
}
