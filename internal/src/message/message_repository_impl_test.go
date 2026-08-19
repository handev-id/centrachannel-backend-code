package message

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"reflect"
	"testing"
)

type captureQueryDriver struct {
	conn *captureQueryConn
}

func (d *captureQueryDriver) Open(string) (driver.Conn, error) {
	d.conn = &captureQueryConn{}
	return d.conn, nil
}

type captureQueryConn struct {
	query string
	args  []interface{}
}

func (c *captureQueryConn) Prepare(string) (driver.Stmt, error) {
	return nil, io.EOF
}

func (c *captureQueryConn) Close() error { return nil }

func (c *captureQueryConn) Begin() (driver.Tx, error) { return nil, io.EOF }

func (c *captureQueryConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.query = query
	c.args = make([]interface{}, len(args))
	for i, a := range args {
		c.args[i] = a.Value
	}
	return &emptyRows{}, nil
}

type emptyRows struct{}

func (r *emptyRows) Columns() []string {
	return []string{"id", "tenant_id", "text", "attachment", "status", "sender_id", "sender_type", "webhook_message_id", "webhook_message_reply_id", "conversation_id", "created_at", "updated_at"}
}

func (r *emptyRows) Close() error { return nil }

func (r *emptyRows) Next(dest []driver.Value) error { return io.EOF }

var captureDriver = &captureQueryDriver{}

func init() {
	for _, name := range sql.Drivers() {
		if name == "capturequery" {
			return
		}
	}
	sql.Register("capturequery", captureDriver)
}

func openCaptureDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("capturequery", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMessageRepository_ListCursor_scopesByTenant(t *testing.T) {
	db := openCaptureDB(t)
	repo := NewMessageRepository()

	t.Run("first page includes tenant_id filter", func(t *testing.T) {
		msgs, err := repo.ListCursor(context.Background(), db, 7, 1, 50, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 0 {
			t.Fatalf("expected 0 messages, got %d", len(msgs))
		}

		if !containsSubstring(captureDriver.conn.query, "tenant_id = $2") {
			t.Errorf("query missing tenant_id filter, got: %s", captureDriver.conn.query)
		}
		if !containsSubstring(captureDriver.conn.query, "conversation_id = $1") {
			t.Errorf("query missing conversation_id filter, got: %s", captureDriver.conn.query)
		}
		wantArgs := []interface{}{int64(1), int64(7), int64(50)}
		if !reflect.DeepEqual(captureDriver.conn.args, wantArgs) {
			t.Errorf("args = %v, want %v", captureDriver.conn.args, wantArgs)
		}
	})

	t.Run("cursor page keeps correct arg order", func(t *testing.T) {
		_, err := repo.ListCursor(context.Background(), db, 7, 1, 50, 99)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !containsSubstring(captureDriver.conn.query, "conversation_id = $1") {
			t.Errorf("query missing conversation_id filter, got: %s", captureDriver.conn.query)
		}
		if !containsSubstring(captureDriver.conn.query, "tenant_id = $2") {
			t.Errorf("query missing tenant_id filter, got: %s", captureDriver.conn.query)
		}
		if !containsSubstring(captureDriver.conn.query, "AND id < $3") {
			t.Errorf("query missing id cursor filter, got: %s", captureDriver.conn.query)
		}
		if !containsSubstring(captureDriver.conn.query, "LIMIT $4") {
			t.Errorf("query missing LIMIT placeholder, got: %s", captureDriver.conn.query)
		}
		wantArgs := []interface{}{int64(1), int64(7), int64(99), int64(50)}
		if !reflect.DeepEqual(captureDriver.conn.args, wantArgs) {
			t.Errorf("args = %v, want %v", captureDriver.conn.args, wantArgs)
		}
	})
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
