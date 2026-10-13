package message

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"centrachannel/config"
	"centrachannel/internal/messenger"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
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

func (c *captureQueryConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.query = query
	c.args = make([]interface{}, len(args))
	for i, arg := range args {
		c.args[i] = arg.Value
	}
	return driver.RowsAffected(1), nil
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
		msgs, err := repo.ListCursor(context.Background(), db, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), 50, uuid.Nil)
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
		wantArgs := []interface{}{"00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000007", int64(50)}
		if !reflect.DeepEqual(captureDriver.conn.args, wantArgs) {
			t.Errorf("args = %v, want %v", captureDriver.conn.args, wantArgs)
		}
	})

	t.Run("cursor page keeps correct arg order", func(t *testing.T) {
		_, err := repo.ListCursor(context.Background(), db, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), 50, uuid.MustParse("00000000-0000-4000-8000-000000000063"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !containsSubstring(captureDriver.conn.query, "conversation_id = $1") {
			t.Errorf("query missing conversation_id filter, got: %s", captureDriver.conn.query)
		}
		if !containsSubstring(captureDriver.conn.query, "tenant_id = $2") {
			t.Errorf("query missing tenant_id filter, got: %s", captureDriver.conn.query)
		}
		if !containsSubstring(captureDriver.conn.query, "AND (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $3 AND tenant_id = $2 AND conversation_id = $1)") {
			t.Errorf("query missing scoped timestamp/UUID cursor filter, got: %s", captureDriver.conn.query)
		}
		if !containsSubstring(captureDriver.conn.query, "LIMIT $4") {
			t.Errorf("query missing LIMIT placeholder, got: %s", captureDriver.conn.query)
		}
		wantArgs := []interface{}{"00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000007", "00000000-0000-4000-8000-000000000063", int64(50)}
		if !reflect.DeepEqual(captureDriver.conn.args, wantArgs) {
			t.Errorf("args = %v, want %v", captureDriver.conn.args, wantArgs)
		}
	})
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func TestMessageRepository_WebhookUpdatesScopeTenant(t *testing.T) {
	tenantID := uuid.New()
	db := openCaptureDB(t)
	repo := NewMessageRepository()
	operations := map[string]func() error{
		"message status": func() error {
			return repo.UpdateStatusByWebhookID(context.Background(), db, tenantID, "shared-provider-id", "delivered")
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			if err := operation(); err != nil {
				t.Fatal(err)
			}
			if !containsSubstring(captureDriver.conn.query, "AND tenant_id=$4") {
				t.Errorf("update must scope provider/message ID by tenant: %s", captureDriver.conn.query)
			}
			args := captureDriver.conn.args
			if len(args) != 4 || args[3] != tenantID.String() {
				t.Errorf("tenant must be passed as fourth argument: %v", args)
			}
		})
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func openMessageTestDB(t *testing.T) (*sql.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("CENTRACHANNEL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set CENTRACHANNEL_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	schema := "message_test_" + uuid.New().String()
	if _, err = db.Exec(`CREATE SCHEMA "` + schema + `"; SET search_path TO "` + schema + `"`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); _ = db.Close() })
	tenantID, convID, profileID, channelID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`CREATE TABLE tenants(id UUID PRIMARY KEY);
      CREATE TABLE profiles(id UUID PRIMARY KEY, tenant_id UUID, external_id TEXT, contact_id UUID, linked_device_whatsapp_id TEXT);
      CREATE TABLE contacts(id UUID PRIMARY KEY, first_name TEXT, last_name TEXT);
      CREATE TABLE users(id UUID PRIMARY KEY, first_name TEXT, last_name TEXT, tenant_id UUID, deleted_at TIMESTAMPTZ);
      CREATE TABLE conversations(id UUID PRIMARY KEY, tenant_id UUID,profile_id UUID,channel_id UUID,last_message JSONB,last_activity TIMESTAMPTZ,unread_count INTEGER DEFAULT 0,last_agent_id UUID,updated_at TIMESTAMPTZ);
      INSERT INTO tenants VALUES ('` + tenantID.String() + `');
      INSERT INTO profiles(id,tenant_id,external_id) VALUES ('` + profileID.String() + `','` + tenantID.String() + `','628123456789');
      INSERT INTO conversations(id,tenant_id,profile_id,channel_id,last_activity) VALUES ('` + convID.String() + `','` + tenantID.String() + `','` + profileID.String() + `','` + channelID.String() + `','2026-10-13T00:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "..", "database", "migrations")
	raw, err := os.ReadFile(filepath.Join(root, "20260614000010_create_messages_table.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(raw)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"20261012000001_add_pending_message_status.up.sql", "20261013000001_message_content_support.up.sql"} {
		raw, err = os.ReadFile(filepath.Join(root, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	return db, tenantID, convID, profileID, channelID
}

func TestMessageRepository_PersistsQuoteSnapshot(t *testing.T) {
	db, tenantID, convID, _, _ := openMessageTestDB(t)
	text, original, providerID := "Reply", "Original", "reply-id"
	incoming := &Message{TenantID: tenantID, ConversationID: convID, Text: &text, Status: "unread", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID, Reply: &Quote{ProviderMessageID: "original-id", SenderType: "user", Text: &original}}
	repo := NewMessageRepository()
	if _, err := createRepositoryTestMessage(t, db, repo, incoming); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ListCursor(context.Background(), db, tenantID, convID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Reply == nil || rows[0].Reply.Text == nil || *rows[0].Reply.Text != "Original" {
		t.Fatalf("quote not persisted: %+v", rows)
	}
	wire, _ := json.Marshal(rows[0])
	var response map[string]interface{}
	_ = json.Unmarshal(wire, &response)
	if response["is_edited"] != false {
		t.Fatalf("read status must not imply an edit: %s", wire)
	}
	if !rows[0].CreatedAt.Before(time.Now().Add(time.Second)) {
		t.Fatal("unexpected created_at")
	}
}

func TestMessageRepository_EditUpdatesOriginalWithoutNewBubble(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openMessageTestDB(t)
	repo := NewMessageRepository()
	original, providerID := "Original", "target-id"
	incoming := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "read", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	id, err := createRepositoryTestMessage(t, db, repo, incoming)
	if err != nil {
		t.Fatal(err)
	}
	before, err := repo.ListCursor(context.Background(), db, tenantID, convID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	edited := "Edited"
	updated, err := NewMessageService(repo, nil, nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil).ApplyMutation(context.Background(), tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Text: &edited, EventID: "edit-id", EventAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil || updated.ID != id || updated.Text == nil || *updated.Text != "Edited" || !updated.IsEdited || updated.EditedAt == nil || updated.Status != "read" {
		t.Fatalf("edit not applied to original: %+v", updated)
	}
	after, err := repo.ListCursor(context.Background(), db, tenantID, convID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || !after[0].CreatedAt.Equal(before[0].CreatedAt) {
		t.Fatal("edit added a bubble or changed original position")
	}
	var unread int
	var activity time.Time
	if err = db.QueryRow(`SELECT unread_count,last_activity FROM conversations WHERE id=$1`, convID).Scan(&unread, &activity); err != nil {
		t.Fatal(err)
	}
	if unread != 0 || !activity.Equal(time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("edit changed unread or last activity")
	}
}

func TestMessageRepository_EditBeforeTargetIsRetained(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openMessageTestDB(t)
	repo := NewMessageRepository()
	edited, original, providerID := "Edited before download finished", "Original", "delayed-target"
	updated, err := NewMessageService(repo, nil, nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil).ApplyMutation(context.Background(), tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Text: &edited, EventID: "edit", EventAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if updated != nil {
		t.Fatal("missing target must not create an edit bubble")
	}
	incoming := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "unread", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	if _, err = createRepositoryTestMessage(t, db, repo, incoming); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ListCursor(context.Background(), db, tenantID, convID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !rows[0].IsEdited || *rows[0].Text != edited {
		t.Fatalf("edit arriving first was lost: %+v", rows)
	}
}

func TestMessageRepository_ReactionStaysOnTarget(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openMessageTestDB(t)
	repo := NewMessageRepository()
	original, providerID := "Original", "reaction-target"
	target := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "read", SenderID: uuid.New(), SenderType: "user", WebhookMessageID: &providerID}
	id, err := createRepositoryTestMessage(t, db, repo, target)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := NewMessageService(repo, nil, nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil).ApplyMutation(context.Background(), tenantID, profileID, channelID, Mutation{Kind: "reaction", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Emoji: "👍", EventID: "reaction", EventAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil || updated.ID != id {
		t.Fatal("reaction has no target")
	}
	wire, _ := json.Marshal(updated)
	var response struct {
		Reactions []struct {
			Emoji      string `json:"emoji"`
			SenderType string `json:"sender_type"`
		} `json:"reactions"`
	}
	_ = json.Unmarshal(wire, &response)
	if len(response.Reactions) != 1 || response.Reactions[0].Emoji != "👍" || response.Reactions[0].SenderType != "contact" {
		t.Fatalf("reaction must belong to target: %s", wire)
	}
	rows, err := repo.ListCursor(context.Background(), db, tenantID, convID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	wire, _ = json.Marshal(rows)
	if len(rows) != 1 {
		t.Fatalf("reaction created a bubble: %s", wire)
	}
	var refresh []struct {
		Reactions []struct {
			Emoji string `json:"emoji"`
		} `json:"reactions"`
	}
	_ = json.Unmarshal(wire, &refresh)
	if len(refresh[0].Reactions) != 1 {
		t.Fatal("reaction lost after refresh")
	}
}

func TestMessageRepository_ReactionRemovalRejectsLateAdd(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openMessageTestDB(t)
	repo := NewMessageRepository()
	text, providerID := "Target", "remove-target"
	target := &Message{TenantID: tenantID, ConversationID: convID, Text: &text, Status: "read", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	if _, err := createRepositoryTestMessage(t, db, repo, target); err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	for _, event := range []Mutation{
		{Kind: "reaction", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Emoji: "👍", EventID: "add", EventAt: at},
		{Kind: "reaction", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Emoji: "", EventID: "remove", EventAt: at.Add(time.Second)},
		{Kind: "reaction", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Emoji: "👍", EventID: "add", EventAt: at},
	} {
		if _, err := NewMessageService(repo, nil, nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil).ApplyMutation(context.Background(), tenantID, profileID, channelID, event); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := repo.ListCursor(context.Background(), db, tenantID, convID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(rows[0].Reactions) != 0 {
		t.Fatalf("removed reaction resurrected: %+v", rows)
	}
}

func TestMessageRepository_EditDoesNotChangeMediaURL(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openMessageTestDB(t)
	repo := NewMessageRepository()
	caption, providerID := "Old caption", "caption-target"
	target := &Message{TenantID: tenantID, ConversationID: convID, Text: &caption, Attachment: json.RawMessage(`{"type":"image","url":"https://storage.example/image.jpg","status":"ready","caption":"Old caption"}`), Status: "read", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	if _, err := createRepositoryTestMessage(t, db, repo, target); err != nil {
		t.Fatal(err)
	}
	edited := "New caption"
	updated, err := NewMessageService(repo, nil, nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil).ApplyMutation(context.Background(), tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Text: &edited, EventID: "caption-edit", EventAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	var att Attachment
	if err = json.Unmarshal(updated.Attachment, &att); err != nil {
		t.Fatal(err)
	}
	if att.URL != "https://storage.example/image.jpg" || att.Caption != edited {
		t.Fatalf("caption edit damaged media: %s", updated.Attachment)
	}
}

func TestMessageRepository_EditBeforeProviderIDIsRetained(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openMessageTestDB(t)
	repo := NewMessageRepository()
	original, edited, providerID := "Original", "Phone edit", "sending-target"
	pending := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "pending", SenderID: uuid.New(), SenderType: "user"}
	if _, err := createRepositoryTestMessage(t, db, repo, pending); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE conversations SET last_message=jsonb_build_object('id',$1::text,'text','Original') WHERE id=$2`, pending.ID.String(), convID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMessageService(repo, nil, nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil).ApplyMutation(context.Background(), tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Text: &edited, EventID: "edit-before-send-response", EventAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateWebhookID(context.Background(), db, tenantID, pending.ID, providerID); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetByID(context.Background(), db, tenantID, convID, pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.IsEdited || *stored.Text != edited {
		t.Fatalf("edit lost during provider ID assignment: %+v", stored)
	}
	var preview string
	if err := db.QueryRow(`SELECT last_message->>'text' FROM conversations WHERE id=$1`, convID).Scan(&preview); err != nil {
		t.Fatal(err)
	}
	if preview != edited {
		t.Fatal("last message preview did not receive retained edit")
	}
	if err := repo.UpdateWebhookID(context.Background(), db, uuid.New(), pending.ID, "wrong-tenant"); err == nil {
		t.Fatal("cross-tenant provider ID update allowed")
	}
}

func createRepositoryTestMessage(t *testing.T, db *sql.DB, repo MessageRepository, msg *Message) (uuid.UUID, error) {
	t.Helper()
	return repo.Create(context.Background(), db, msg, nil, nil, nil)
}

// mock SQL driver for creating *sql.DB without a real database

func init() {
	for _, name := range sql.Drivers() {
		if name == "mock" {
			return
		}
	}
	sql.Register("mock", &mockSQLDriver{})
}

type mockSQLDriver struct{}

func (d *mockSQLDriver) Open(string) (driver.Conn, error) {
	return &mockSQLConn{}, nil
}

type mockSQLConn struct{}

func (c *mockSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *mockSQLConn) Close() error {
	return nil
}

func (c *mockSQLConn) Begin() (driver.Tx, error) {
	return &mockSQLTx{}, nil
}

type mockSQLTx struct{}

func (t *mockSQLTx) Commit() error   { return nil }
func (t *mockSQLTx) Rollback() error { return nil }

// mock repositories

type mockMessageRepository struct {
	getByIDFunc func(context.Context, DBTX, uuid.UUID, uuid.UUID, uuid.UUID) (*Message, error)
	MessageRepository
	createFunc                  func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error)
	listCursorFunc              func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error)
	updateStatusFunc            func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error
	updateStatusByWebhookIDFunc func(ctx context.Context, q DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error
}

func (m *mockMessageRepository) GetByID(ctx context.Context, q DBTX, tenantID, conversationID, id uuid.UUID) (*Message, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, tenantID, conversationID, id)
	}
	return nil, fmt.Errorf("message not found")
}

func (m *mockMessageRepository) Create(ctx context.Context, db *sql.DB, msg *Message, lastMessageJSON []byte, lastAgentID *uuid.UUID, conversations conversation.ConversationRepository) (uuid.UUID, error) {

	id, err := m.createFunc(ctx, db, msg)
	if err != nil {
		return uuid.Nil, err
	}
	if conversations != nil {
		if err := conversations.UpdateLastMessage(ctx, db, msg.TenantID, msg.ConversationID, lastMessageJSON, lastAgentID); err != nil {
			return uuid.Nil, err
		}
	}
	return id, nil

}

func (m *mockMessageRepository) ListCursor(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error) {
	return m.listCursorFunc(ctx, q, tenantID, conversationID, limit, lastID)
}

func (m *mockMessageRepository) UpdateStatus(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	return m.updateStatusFunc(ctx, q, tenantID, id, status)
}

func (m *mockMessageRepository) UpdateStatusByWebhookID(ctx context.Context, q DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
	if m.updateStatusByWebhookIDFunc != nil {
		return m.updateStatusByWebhookIDFunc(ctx, q, tenantID, webhookMessageID, status)
	}
	return nil
}

func (m *mockMessageRepository) UpdateWebhookID(ctx context.Context, db *sql.DB, tenantID uuid.UUID, id uuid.UUID, webhookMessageID string) error {
	return nil
}

func (m *mockMessageRepository) MarkReadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) error {
	return nil
}

func (m *mockMessageRepository) GetUnreadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]*Message, error) {
	return nil, nil
}

type mockConversationRepository struct {
	updateLastMessageFunc func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error
}

func (m *mockConversationRepository) GetByID(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) (*conversation.Conversation, error) {
	return &conversation.Conversation{ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), ChannelID: uuid.MustParse("00000000-0000-4000-8000-000000000001")}, nil
}

func (m *mockConversationRepository) Create(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (uuid.UUID, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) UpdateStatus(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) Assign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, agentID uuid.UUID) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) Unassign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) MarkRead(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) GetTotalUnread(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID) (int, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) ListCursor(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, limit int, status string, channelID, agentID uuid.UUID, search string, lastActivity *time.Time, lastID uuid.UUID) ([]*conversation.Conversation, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) UpdateLastMessage(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
	return m.updateLastMessageFunc(ctx, q, tenantID, id, lastMessageJSON, lastAgentID)
}

func (m *mockConversationRepository) FindOpenByProfileAndChannel(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
	return nil, nil
}

// mockProfileRepository
type mockProfileRepository struct {
	getByExternalIDAndChannelIDFunc func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error)
	createFunc                      func(ctx context.Context, q profile.DBTX, profile *profile.Profile) (uuid.UUID, error)
}

func (m *mockProfileRepository) GetByContactIDs(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID][]profile.Profile, error) {
	return map[uuid.UUID][]profile.Profile{}, nil
}
func (m *mockProfileRepository) List(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID, channelID uuid.UUID) ([]profile.Profile, error) {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID) (*profile.Profile, error) {
	return &profile.Profile{ExternalID: "test_external_id"}, nil
}
func (m *mockProfileRepository) Update(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID, p *profile.Profile) error {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByContactID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID uuid.UUID) ([]profile.Profile, error) {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByExternalIDAndChannelID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
	if m.getByExternalIDAndChannelIDFunc != nil {
		return m.getByExternalIDAndChannelIDFunc(ctx, q, tenantID, externalID, channelID)
	}
	return nil, nil
}
func (m *mockProfileRepository) Create(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, p)
	}
	return uuid.Nil, nil
}

// mockChannelRepository
type mockChannelRepository struct {
	getByIDFunc   func(context.Context, channel.DBTX, uuid.UUID) (*channel.Channel, error)
	getByTypeFunc func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error)
}

func (m *mockChannelRepository) List(ctx context.Context, q channel.DBTX) ([]channel.Channel, error) {
	panic("unexpected call")
}
func (m *mockChannelRepository) GetByID(ctx context.Context, q channel.DBTX, id uuid.UUID) (*channel.Channel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, id)
	}
	return &channel.Channel{Type: "facebook"}, nil
}
func (m *mockChannelRepository) GetByType(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
	if m.getByTypeFunc != nil {
		return m.getByTypeFunc(ctx, q, channelType)
	}
	return nil, nil
}

// mockTenantRepository
type mockTenantRepository struct{}

func (m *mockTenantRepository) Create(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) (uuid.UUID, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) CreateRole(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) CreateUser(ctx context.Context, q tenant.DBTX, user *tenant.User) (uuid.UUID, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) AttachRole(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error {
	panic("unexpected call")
}
func (m *mockTenantRepository) List(ctx context.Context, q tenant.DBTX) ([]tenant.Tenant, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) GetByID(ctx context.Context, q tenant.DBTX, id uuid.UUID) (*tenant.Tenant, error) {
	return &tenant.Tenant{}, nil
}
func (m *mockTenantRepository) GetByMetaPageID(ctx context.Context, q tenant.DBTX, pageID string) (*tenant.Tenant, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) GetByMetaInstagramBusinessID(ctx context.Context, q tenant.DBTX, igID string) (*tenant.Tenant, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) Update(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) error {
	panic("unexpected call")
}

// mockMessenger
type mockMessenger struct{}

func (m *mockMessenger) Send(msg *messenger.OutgoingMessage) (string, error) {
	return "ext_msg_123", nil
}

// helpers

func strPtr(s string) *string { return &s }

type notifiedEvent struct {
	tenantID uuid.UUID
	event    string
	data     interface{}
}

type mockNotifier struct {
	events []notifiedEvent
}

func (m *mockNotifier) Notify(tenantID uuid.UUID, event string, data interface{}) {
	m.events = append(m.events, notifiedEvent{tenantID: tenantID, event: event, data: data})
}

func (m *mockNotifier) last() *notifiedEvent {
	if len(m.events) == 0 {
		return nil
	}
	return &m.events[len(m.events)-1]
}

// tests

func TestMessageService_Send(t *testing.T) {
	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	t.Run("sender type user passes lastAgentID to UpdateLastMessage", func(t *testing.T) {
		var capturedCreateMsg *Message
		var capturedLastAgentID *uuid.UUID

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				capturedCreateMsg = msg
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastAgentID = lastAgentID
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("hello"),
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-00000000002a"),
			SenderType: "user",
		}
		msg, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected ID 1, got %s", msg.ID)
		}
		if capturedCreateMsg == nil {
			t.Fatal("expected Create to be called")
		}
		if capturedCreateMsg.TenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected TenantID 1, got %s", capturedCreateMsg.TenantID)
		}
		if capturedCreateMsg.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
			t.Errorf("expected ConversationID 10, got %s", capturedCreateMsg.ConversationID)
		}
		if capturedCreateMsg.SenderType != "user" {
			t.Errorf("expected SenderType user, got %s", capturedCreateMsg.SenderType)
		}
		if capturedCreateMsg.SenderID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
			t.Errorf("expected SenderID 42, got %s", capturedCreateMsg.SenderID)
		}
		if capturedCreateMsg.Status != "pending" {
			t.Errorf("expected Status pending, got %s", capturedCreateMsg.Status)
		}
		if capturedLastAgentID == nil || *capturedLastAgentID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
			t.Errorf("expected lastAgentID 42 for user sender, got %v", capturedLastAgentID)
		}
	})

	t.Run("sender type contact passes nil as lastAgentID", func(t *testing.T) {
		var capturedLastAgentID *uuid.UUID

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastAgentID = lastAgentID
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("hello"),
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-000000000063"),
			SenderType: "contact",
		}
		_, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLastAgentID != nil {
			t.Errorf("expected nil lastAgentID for contact sender, got %s", *capturedLastAgentID)
		}
	})

	t.Run("attachment is included in last_message JSON when present", func(t *testing.T) {
		var capturedLastMsgJSON []byte

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastMsgJSON = lastMessageJSON
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("with attachment"),
			Attachment: json.RawMessage(`{"url":"http://example.com/file.pdf","type":"document","mimetype":"application/pdf","size":100}`),
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			SenderType: "user",
		}
		_, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(capturedLastMsgJSON, &result); err != nil {
			t.Fatalf("failed to unmarshal lastMsgJSON: %v", err)
		}
		att, ok := result["attachment"]
		if !ok {
			t.Fatal("expected attachment key in last_message JSON")
		}
		attMap, ok := att.(map[string]interface{})
		if !ok {
			t.Fatal("expected attachment to be an object")
		}
		if attMap["url"] != "http://example.com/file.pdf" {
			t.Errorf("expected url http://example.com/file.pdf, got %v", attMap["url"])
		}
		if attMap["type"] != "document" {
			t.Errorf("expected type document, got %v", attMap["type"])
		}
	})

	t.Run("attachment omitted from last_message JSON when absent", func(t *testing.T) {
		var capturedLastMsgJSON []byte

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastMsgJSON = lastMessageJSON
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("no attachment"),
			Attachment: nil,
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			SenderType: "user",
		}
		_, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(capturedLastMsgJSON, &result); err != nil {
			t.Fatalf("failed to unmarshal lastMsgJSON: %v", err)
		}
		if _, ok := result["attachment"]; ok {
			t.Error("attachment should not appear in last_message when absent from request")
		}
	})
}

func TestMessageService_Send_NotifiesNewMessage(t *testing.T) {
	notifier := &mockNotifier{}

	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	msgRepo := &mockMessageRepository{
		createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
		},
	}
	convRepo := &mockConversationRepository{
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}

	svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, notifier)
	req := SendMessageRequest{
		Text:       strPtr("hello"),
		SenderID:   uuid.MustParse("00000000-0000-4000-8000-00000000002a"),
		SenderType: "user",
	}
	msg, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected ID 1, got %s", msg.ID)
	}

	ev := notifier.last()
	if ev == nil {
		t.Fatal("expected a websocket notification")
	}
	if ev.tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000007") {
		t.Errorf("expected tenantID 7, got %s", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	notified, ok := ev.data.(*Message)
	if !ok {
		t.Fatalf("expected data *Message, got %T", ev.data)
	}
	if notified.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected message id 1, got %s", notified.ID)
	}
	if notified.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
		t.Errorf("expected conversation_id 10, got %s", notified.ConversationID)
	}
	if notified.SenderType != "user" {
		t.Errorf("expected sender_type user, got %s", notified.SenderType)
	}
}

func TestMessageService_ListCursor(t *testing.T) {
	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	t.Run("defaults_limit_and_passes_cursor", func(t *testing.T) {
		var capturedLimit int
		var capturedLastID uuid.UUID
		var capturedTenantID uuid.UUID

		msgRepo := &mockMessageRepository{
			listCursorFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error) {
				capturedTenantID = tenantID
				capturedLimit = limit
				capturedLastID = lastID
				return []*Message{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000032")}, {ID: uuid.MustParse("00000000-0000-4000-8000-000000000031")}, {ID: uuid.MustParse("00000000-0000-4000-8000-000000000030")}}, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		result, lastID, hasMore, err := svc.ListCursor(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), ListMessageQuery{Limit: 0, LastID: uuid.MustParse("00000000-0000-4000-8000-000000000037")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000007") {
			t.Errorf("expected tenantID 7, got %s", capturedTenantID)
		}
		if capturedLimit != 51 {
			t.Errorf("expected limit+1 = 51, got %d", capturedLimit)
		}
		if capturedLastID != uuid.MustParse("00000000-0000-4000-8000-000000000037") {
			t.Errorf("expected lastID 55, got %s", capturedLastID)
		}
		if hasMore {
			t.Error("expected hasMore false")
		}
		// repo returns DESC (newest first); service reverses to ascending chronological order
		if len(result) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(result))
		}
		if result[0].ID != uuid.MustParse("00000000-0000-4000-8000-000000000030") || result[1].ID != uuid.MustParse("00000000-0000-4000-8000-000000000031") || result[2].ID != uuid.MustParse("00000000-0000-4000-8000-000000000032") {
			t.Errorf("expected reversed ascending order [48,49,50], got [%s,%s,%s]", result[0].ID, result[1].ID, result[2].ID)
		}
		if lastID != uuid.MustParse("00000000-0000-4000-8000-000000000030") {
			t.Errorf("expected cursor lastID 48 (oldest kept), got %s", lastID)
		}
	})

	t.Run("has_more_trims_and_keeps_newest", func(t *testing.T) {
		var capturedLimit int
		msgs := make([]*Message, 0, 51)
		for i := 51; i >= 1; i-- {
			msgs = append(msgs, &Message{ID: uuid.UUID{6: 0x40, 8: 0x80, 15: byte(i)}})
		}

		msgRepo := &mockMessageRepository{
			listCursorFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error) {
				capturedLimit = limit
				return msgs, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		result, lastID, hasMore, err := svc.ListCursor(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), ListMessageQuery{Limit: 50, LastID: uuid.MustParse("00000000-0000-4000-8000-00000000003c")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLimit != 51 {
			t.Errorf("expected limit+1 = 51, got %d", capturedLimit)
		}
		if !hasMore {
			t.Error("expected hasMore true")
		}
		if len(result) != 50 {
			t.Fatalf("expected 50 messages, got %d", len(result))
		}
		if result[0].ID != uuid.MustParse("00000000-0000-4000-8000-000000000002") || result[49].ID != uuid.MustParse("00000000-0000-4000-8000-000000000033") {
			t.Errorf("expected ascending [2..51], got first=%s last=%s", result[0].ID, result[49].ID)
		}
		if lastID != uuid.MustParse("00000000-0000-4000-8000-000000000002") {
			t.Errorf("expected cursor lastID 2 (oldest kept), got %s", lastID)
		}
	})
}

func TestMessageService_UpdateStatus(t *testing.T) {
	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	t.Run("delegates to repository", func(t *testing.T) {
		var capturedID uuid.UUID
		var capturedStatus string

		msgRepo := &mockMessageRepository{
			updateStatusFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
				capturedID = id
				capturedStatus = status
				return nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		if err := svc.UpdateStatus(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), "read"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Errorf("expected id 5, got %s", capturedID)
		}
		if capturedStatus != "read" {
			t.Errorf("expected status 'read', got %s", capturedStatus)
		}
	})
}

func (m *mockTenantRepository) GetDomains(ctx context.Context, q tenant.DBTX, id uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage("[]"), nil
}
func (m *mockTenantRepository) GetActiveIDByDomain(ctx context.Context, q tenant.DBTX, name string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockMessageRepository) RetryFailed(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, senderID uuid.UUID) (*Message, error) {
	return nil, fmt.Errorf("not found")
}

func TestSendReplyUsesScopedTarget(t *testing.T) {
	tenantID, conversationID, targetID := uuid.New(), uuid.New(), uuid.New()
	providerID, original := "original-provider", "Original"
	repo := &mockMessageRepository{createFunc: func(_ context.Context, _ DBTX, msg *Message) (uuid.UUID, error) { return uuid.New(), nil }, getByIDFunc: func(_ context.Context, _ DBTX, tenant, conv, id uuid.UUID) (*Message, error) {
		if tenant != tenantID || conv != conversationID || id != targetID {
			t.Fatal("reply target lookup not scoped")
		}
		return &Message{ID: targetID, TenantID: tenantID, ConversationID: conversationID, WebhookMessageID: &providerID, Text: &original, SenderType: "contact"}, nil
	}}
	channels := &mockChannelRepository{getByIDFunc: func(context.Context, channel.DBTX, uuid.UUID) (*channel.Channel, error) {
		return &channel.Channel{Type: "whatsapp"}, nil
	}}
	repo.updateStatusFunc = func(context.Context, DBTX, uuid.UUID, uuid.UUID, string) error { return nil }
	convs := &mockConversationRepository{updateLastMessageFunc: func(context.Context, conversation.DBTX, uuid.UUID, uuid.UUID, []byte, *uuid.UUID) error { return nil }}
	svc := NewMessageService(repo, convs, &mockProfileRepository{}, channels, &mockTenantRepository{}, nil, &config.Config{}, logger.NewLogger("error", "text"), nil)
	req := SendMessageRequest{Text: strPtr("Reply"), SenderID: uuid.New(), SenderType: "user"}
	if err := json.Unmarshal([]byte(`{"reply_to_id":"`+targetID.String()+`"}`), &req); err != nil {
		t.Fatal(err)
	}
	sent, err := svc.Send(context.Background(), req, tenantID, conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if sent.Reply == nil || sent.Reply.ProviderMessageID != providerID || sent.Reply.MessageID == nil || *sent.Reply.MessageID != targetID {
		t.Fatalf("sent reply lacks original context: %+v", sent)
	}
}

func TestCreateRollsBackMessageWhenConversationPreviewFails(t *testing.T) {
	db, tenantID, conversationID, _, _ := openMessageTestDB(t)
	repo := NewMessageRepository()
	service := NewMessageService(repo, &mockConversationRepository{updateLastMessageFunc: func(context.Context, conversation.DBTX, uuid.UUID, uuid.UUID, []byte, *uuid.UUID) error {
		return fmt.Errorf("preview failed")
	}}, nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil)
	text, providerID := "Message", "atomic-create"
	incoming := &Message{TenantID: tenantID, ConversationID: conversationID, Text: &text, SenderType: "contact", SenderID: uuid.New(), Status: "unread", WebhookMessageID: &providerID}
	if _, err := service.Create(context.Background(), incoming, nil); err == nil {
		t.Fatal("expected preview failure")
	}
	rows, err := repo.ListCursor(context.Background(), db, tenantID, conversationID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatal("message persisted after preview transaction failed")
	}
}

func TestCreateDuplicateDoesNotRepeatUnreadCount(t *testing.T) {
	db, tenantID, conversationID, _, _ := openMessageTestDB(t)
	repo := NewMessageRepository()
	service := NewMessageService(repo, conversation.NewConversationRepository(), nil, nil, nil, db, nil, logger.NewLogger("error", "text"), nil)
	text, providerID := "Message", "duplicate-create"
	for i := 0; i < 2; i++ {
		incoming := &Message{TenantID: tenantID, ConversationID: conversationID, Text: &text, SenderType: "contact", SenderID: uuid.New(), Status: "unread", WebhookMessageID: &providerID}
		id, err := service.Create(context.Background(), incoming, nil)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 && id == uuid.Nil {
			t.Fatal("original message not created")
		}
		if i == 1 && id != uuid.Nil {
			t.Fatal("duplicate created a second message")
		}
	}
	var unread int
	if err := db.QueryRow(`SELECT unread_count FROM conversations WHERE id=$1`, conversationID).Scan(&unread); err != nil {
		t.Fatal(err)
	}
	if unread != 1 {
		t.Fatalf("duplicate changed unread count: %d", unread)
	}
}
