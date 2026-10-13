package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func openContentTestDB(t *testing.T) (*sql.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
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

func TestContentRepository_PersistsQuoteSnapshot(t *testing.T) {
	db, tenantID, convID, _, _ := openContentTestDB(t)
	text, original, providerID := "Reply", "Original", "reply-id"
	incoming := &Message{TenantID: tenantID, ConversationID: convID, Text: &text, Status: "unread", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID, Reply: &Quote{ProviderMessageID: "original-id", SenderType: "user", Text: &original}}
	repo := NewMessageRepository()
	if _, err := repo.Create(context.Background(), db, incoming, nil, nil, nil); err != nil {
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

func TestContentRepository_EditUpdatesOriginalWithoutNewBubble(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openContentTestDB(t)
	repo := NewMessageRepository()
	original, providerID := "Original", "target-id"
	incoming := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "read", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	id, err := repo.Create(context.Background(), db, incoming, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := repo.ListCursor(context.Background(), db, tenantID, convID, 30, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	edited := "Edited"
	updated, err := repo.ApplyMutation(context.Background(), db, tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Text: &edited, EventID: "edit-id", EventAt: time.Now()})
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

func TestContentRepository_EditBeforeTargetIsRetained(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openContentTestDB(t)
	repo := NewMessageRepository()
	edited, original, providerID := "Edited before download finished", "Original", "delayed-target"
	updated, err := repo.ApplyMutation(context.Background(), db, tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Text: &edited, EventID: "edit", EventAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if updated != nil {
		t.Fatal("missing target must not create an edit bubble")
	}
	incoming := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "unread", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	if _, err = repo.Create(context.Background(), db, incoming, nil, nil, nil); err != nil {
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

func TestContentRepository_ReactionStaysOnTarget(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openContentTestDB(t)
	repo := NewMessageRepository()
	original, providerID := "Original", "reaction-target"
	target := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "read", SenderID: uuid.New(), SenderType: "user", WebhookMessageID: &providerID}
	id, err := repo.Create(context.Background(), db, target, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := repo.ApplyMutation(context.Background(), db, tenantID, profileID, channelID, Mutation{Kind: "reaction", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Emoji: "👍", EventID: "reaction", EventAt: time.Now()})
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

func TestContentRepository_ReactionRemovalRejectsLateAdd(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openContentTestDB(t)
	repo := NewMessageRepository()
	text, providerID := "Target", "remove-target"
	target := &Message{TenantID: tenantID, ConversationID: convID, Text: &text, Status: "read", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	if _, err := repo.Create(context.Background(), db, target, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	for _, event := range []Mutation{
		{Kind: "reaction", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Emoji: "👍", EventID: "add", EventAt: at},
		{Kind: "reaction", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Emoji: "", EventID: "remove", EventAt: at.Add(time.Second)},
		{Kind: "reaction", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Emoji: "👍", EventID: "add", EventAt: at},
	} {
		if _, err := repo.ApplyMutation(context.Background(), db, tenantID, profileID, channelID, event); err != nil {
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

func TestContentRepository_EditDoesNotChangeMediaURL(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openContentTestDB(t)
	repo := NewMessageRepository()
	caption, providerID := "Old caption", "caption-target"
	target := &Message{TenantID: tenantID, ConversationID: convID, Text: &caption, Attachment: json.RawMessage(`{"type":"image","url":"https://storage.example/image.jpg","status":"ready","caption":"Old caption"}`), Status: "read", SenderID: uuid.New(), SenderType: "contact", WebhookMessageID: &providerID}
	if _, err := repo.Create(context.Background(), db, target, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	edited := "New caption"
	updated, err := repo.ApplyMutation(context.Background(), db, tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "contact", SenderType: "contact", Text: &edited, EventID: "caption-edit", EventAt: time.Now()})
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

func TestContentRepository_EditBeforeProviderIDIsRetained(t *testing.T) {
	db, tenantID, convID, profileID, channelID := openContentTestDB(t)
	repo := NewMessageRepository()
	original, edited, providerID := "Original", "Phone edit", "sending-target"
	pending := &Message{TenantID: tenantID, ConversationID: convID, Text: &original, Status: "pending", SenderID: uuid.New(), SenderType: "user"}
	if _, err := repo.Create(context.Background(), db, pending, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE conversations SET last_message=jsonb_build_object('id',$1::text,'text','Original') WHERE id=$2`, pending.ID.String(), convID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ApplyMutation(context.Background(), db, tenantID, profileID, channelID, Mutation{Kind: "edit", TargetProviderID: providerID, ActorKey: "user", SenderType: "user", Text: &edited, EventID: "edit-before-send-response", EventAt: time.Now()}); err != nil {
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
