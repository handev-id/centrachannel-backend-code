package database_test

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/campaign"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/dashboard"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/note"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/utils/logger"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Each test uses its own schema. TEST_DATABASE_URL must point to a test database
// whose user can create schemas; the default application database is never used.
func uuidTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL UUID regressions")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	schema := "uuid_test_" + uuid.New().String()
	quoted := `"` + schema + `"`
	if _, err := db.Exec("CREATE SCHEMA " + quoted); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DROP SCHEMA " + quoted + " CASCADE"); err != nil {
			t.Error(err)
		}
		db.Close()
	})
	if _, err := db.Exec("SET search_path TO " + quoted); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("migration files: %v", err)
	}
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(body)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	return db
}

func insertUUID(t *testing.T, db *sql.DB, query string, args ...any) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := db.QueryRow(query, args...).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func testTenant(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	return insertUUID(t, db, `INSERT INTO tenants (name, domain) VALUES ('test', $1) RETURNING id`, uuid.New().String())
}

func testContact(t *testing.T, db *sql.DB, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return insertUUID(t, db, `INSERT INTO contacts (tenant_id, first_name) VALUES ($1, 'test') RETURNING id`, tenantID)
}

func testProfile(t *testing.T, db *sql.DB, tenantID, contactID, channelID uuid.UUID) uuid.UUID {
	t.Helper()
	return insertUUID(t, db, `INSERT INTO profiles (tenant_id, contact_id, channel_id, external_id) VALUES ($1,$2,$3,$4) RETURNING id`, tenantID, contactID, channelID, uuid.New().String())
}

func contactService(db *sql.DB) contact.ContactService {
	return contact.NewContactService(contact.NewContactRepository(), profile.NewProfileRepository(), db, &config.Config{}, logger.NewLogger("error", "text"))
}

func TestUUIDContactMergeRejectsForeignTarget(t *testing.T) {
	db := uuidTestDB(t)
	tenantID := testTenant(t, db)
	source := testContact(t, db, tenantID)
	target := testContact(t, db, testTenant(t, db))
	if err := contactService(db).Merge(context.Background(), tenantID, source, target); err == nil {
		t.Fatal("merge accepted a target from another tenant")
	}
	var active bool
	if err := db.QueryRow(`SELECT deleted_at IS NULL AND merged_to_id IS NULL FROM contacts WHERE id=$1`, source).Scan(&active); err != nil || !active {
		t.Fatalf("rejected merge changed source: active=%v, err=%v", active, err)
	}
}

func TestUUIDContactMergeRoundTrip(t *testing.T) {
	db := uuidTestDB(t)
	tenantID := testTenant(t, db)
	source, target := testContact(t, db, tenantID), testContact(t, db, tenantID)
	channelID := insertUUID(t, db, `INSERT INTO channels (name, type) VALUES ('test', 'whatsapp') RETURNING id`)
	sourceProfile := testProfile(t, db, tenantID, source, channelID)
	targetProfile := testProfile(t, db, tenantID, target, channelID)
	svc := contactService(db)
	if err := svc.Merge(context.Background(), tenantID, source, target); err != nil {
		t.Fatal(err)
	}
	if err := svc.Unmerge(context.Background(), tenantID, source); err != nil {
		t.Fatal(err)
	}
	for profileID, wantContact := range map[uuid.UUID]uuid.UUID{sourceProfile: source, targetProfile: target} {
		var gotContact uuid.UUID
		var provenance uuid.NullUUID
		if err := db.QueryRow(`SELECT contact_id, merged_from_contact_id FROM profiles WHERE id=$1`, profileID).Scan(&gotContact, &provenance); err != nil {
			t.Fatal(err)
		}
		if gotContact != wantContact || provenance.Valid {
			t.Errorf("profile %s: owner=%s, want=%s, provenance=%v", profileID, gotContact, wantContact, provenance)
		}
	}
}

func TestUUIDContactMergeRejectsUnavailableContacts(t *testing.T) {
	db := uuidTestDB(t)
	tenantID := testTenant(t, db)
	for _, scenario := range []string{"missing source", "missing target", "deleted target", "foreign source"} {
		t.Run(scenario, func(t *testing.T) {
			source, target := testContact(t, db, tenantID), testContact(t, db, tenantID)
			switch scenario {
			case "missing source":
				source = uuid.New()
			case "missing target":
				target = uuid.New()
			case "deleted target":
				if _, err := db.Exec(`UPDATE contacts SET deleted_at=NOW() WHERE id=$1`, target); err != nil {
					t.Fatal(err)
				}
			case "foreign source":
				source = testContact(t, db, testTenant(t, db))
			}
			if err := contactService(db).Merge(context.Background(), tenantID, source, target); err == nil {
				t.Fatal("merge accepted unavailable contact")
			}
		})
	}
}

func TestUUIDContactUnmergeRejectsOrdinaryDeletedContact(t *testing.T) {
	db := uuidTestDB(t)
	tenantID := testTenant(t, db)
	id := testContact(t, db, tenantID)
	if _, err := db.Exec(`UPDATE contacts SET deleted_at=NOW() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if err := contactService(db).Unmerge(context.Background(), tenantID, id); err == nil {
		t.Fatal("unmerge resurrected an ordinary deleted contact")
	}
}

func TestUUIDContactMergeRejectsNestedProvenance(t *testing.T) {
	db := uuidTestDB(t)
	tenantID := testTenant(t, db)
	source, target, third := testContact(t, db, tenantID), testContact(t, db, tenantID), testContact(t, db, tenantID)
	svc := contactService(db)
	if err := svc.Merge(context.Background(), tenantID, source, target); err != nil {
		t.Fatal(err)
	}
	if err := svc.Merge(context.Background(), tenantID, target, third); err == nil {
		t.Fatal("nested merge overwrote original provenance")
	}
	if err := svc.Unmerge(context.Background(), tenantID, source); err != nil {
		t.Fatal(err)
	}
	if err := svc.Merge(context.Background(), tenantID, target, third); err != nil {
		t.Fatalf("merge after undoing incoming merge: %v", err)
	}
}

func TestUUIDConversationRejectsForeignAgent(t *testing.T) {
	db := uuidTestDB(t)
	tenantID, otherTenant := testTenant(t, db), testTenant(t, db)
	contactID := testContact(t, db, tenantID)
	channelID := insertUUID(t, db, `INSERT INTO channels (name, type) VALUES ('test', 'whatsapp') RETURNING id`)
	profileID := testProfile(t, db, tenantID, contactID, channelID)
	agentID := insertUUID(t, db, `INSERT INTO users (tenant_id, first_name, username, email, password) VALUES ($1, 'private name', 'agent', 'agent@test', 'hash') RETURNING id`, otherTenant)
	repo := conversation.NewConversationRepository()
	conv := &conversation.Conversation{TenantID: tenantID, ProfileID: profileID, ChannelID: channelID, Status: "assigned", AgentID: &agentID}
	if _, err := repo.Create(context.Background(), db, conv); err == nil {
		t.Error("conversation create accepted an agent from another tenant")
	}
	conv.AgentID, conv.Status = nil, "unassigned"
	convID, err := repo.Create(context.Background(), db, conv)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Assign(context.Background(), db, tenantID, convID, agentID); err == nil {
		t.Error("conversation assign accepted an agent from another tenant")
	}
	if err := repo.UpdateLastMessage(context.Background(), db, tenantID, convID, []byte(`{"text":"test"}`), &agentID); err == nil {
		t.Error("last message accepted an agent from another tenant")
	}
	// Simulate inconsistent data written by the old implementation. The read
	// must defend its tenant boundary even when legacy rows are inconsistent.
	if _, err := db.Exec(`UPDATE conversations SET agent_id=$1 WHERE id=$2`, agentID, convID); err != nil {
		t.Fatal(err)
	}
	items, err := dashboard.NewDashboardRepository().GetRecentConversations(context.Background(), db, tenantID, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Agent != nil && item.Agent.ID == agentID {
			t.Error("dashboard exposed an agent from another tenant")
		}
	}
	localAgent := insertUUID(t, db, `INSERT INTO users (tenant_id, first_name, username, email, password) VALUES ($1, 'local', 'local', 'local@test', 'hash') RETURNING id`, tenantID)
	if err := repo.Assign(context.Background(), db, tenantID, convID, localAgent); err != nil {
		t.Fatalf("live local agent rejected: %v", err)
	}
	if _, err := db.Exec(`UPDATE users SET deleted_at=NOW() WHERE id=$1`, localAgent); err != nil {
		t.Fatal(err)
	}
	if err := repo.Assign(context.Background(), db, tenantID, convID, localAgent); err == nil {
		t.Error("assign accepted a deleted agent")
	}
	conv.AgentID = &localAgent
	if _, err := repo.Create(context.Background(), db, conv); err == nil {
		t.Error("create accepted a deleted agent")
	}
}

func TestUUIDMigrationRoundTrip(t *testing.T) {
	db := uuidTestDB(t)
	var schema string
	if err := db.QueryRow(`SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	// seed.sql uses psql includes relative to the repository root.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	seed := exec.CommandContext(ctx, "psql", "-X", "--set=ON_ERROR_STOP=1", "--single-transaction", "--dbname="+os.Getenv("TEST_DATABASE_URL"), "--file=database/seed.sql")
	seed.Dir = ".."
	seed.Env = append(os.Environ(), `PGOPTIONS=-c search_path="`+schema+`"`)
	if output, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("UUID seed: %v\n%s", err, output)
	}
	var invalid int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=current_schema() AND (column_name='id' OR column_name LIKE '%\_id' ESCAPE '\' OR column_name='created_by') AND column_name NOT IN ('external_id','linked_device_whatsapp_id','whatsapp_id','webhook_message_id','webhook_message_reply_id','target_provider_id','event_id') AND udt_name != 'uuid'`).Scan(&invalid); err != nil || invalid != 0 {
		t.Fatalf("non-UUID internal IDs=%d, err=%v", invalid, err)
	}
	files, err := filepath.Glob("migrations/*.down.sql")
	upFiles, upErr := filepath.Glob("migrations/*.up.sql")
	if err != nil || upErr != nil || len(files) == 0 || len(files) != len(upFiles) {
		t.Fatalf("down migration pairs=%d, err=%v", len(files), err)
	}
	for i := len(files) - 1; i >= 0; i-- {
		body, err := os.ReadFile(files[i])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(body)); err != nil {
			t.Fatalf("%s: %v", files[i], err)
		}
	}
	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=current_schema()`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("tables remaining after rollback=%d, err=%v", remaining, err)
	}
}

func TestUUIDCampaignRejectsForeignRecipientList(t *testing.T) {
	db := uuidTestDB(t)
	tenantID, foreign := testTenant(t, db), testTenant(t, db)
	listID := insertUUID(t, db, `INSERT INTO campaign_recipient_lists (tenant_id, name) VALUES ($1, 'foreign') RETURNING id`, foreign)
	svc := campaign.NewCampaignService(campaign.NewCampaignRepository(), nil, nil, db, &config.Config{}, logger.NewLogger("error", "text"))
	if _, err := svc.AddRecipientContact(context.Background(), tenantID, listID, campaign.AddContactToListRequest{FirstName: "injected"}); err == nil {
		t.Fatal("campaign accepted foreign recipient list")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM campaign_recipient_contacts WHERE campaign_recipient_list_id=$1`, listID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("foreign list mutated: count=%d, err=%v", count, err)
	}
}

func TestUUIDNoteTenantAndPartialUpdate(t *testing.T) {
	db := uuidTestDB(t)
	tenantID, foreign := testTenant(t, db), testTenant(t, db)
	contactID := testContact(t, db, tenantID)
	channelID := insertUUID(t, db, `INSERT INTO channels (name, type) VALUES ('test', 'whatsapp') RETURNING id`)
	profileID := testProfile(t, db, tenantID, contactID, channelID)
	convID := insertUUID(t, db, `INSERT INTO conversations (tenant_id, profile_id, channel_id, status) VALUES ($1,$2,$3,'unassigned') RETURNING id`, tenantID, profileID, channelID)
	repo := note.NewNoteRepository()
	if _, err := repo.Create(context.Background(), db, &note.Note{TenantID: foreign, ConversationID: convID, Text: "foreign"}); err == nil {
		t.Error("note accepted a foreign conversation")
	}
	foreignUser := insertUUID(t, db, `INSERT INTO users (tenant_id, first_name, username, email, password) VALUES ($1, 'private', 'private', 'private@test', 'hash') RETURNING id`, foreign)
	if _, err := repo.Create(context.Background(), db, &note.Note{TenantID: tenantID, ConversationID: convID, UserID: &foreignUser, Text: "foreign"}); err == nil {
		t.Error("note accepted a foreign user")
	}
	date := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	id, err := repo.Create(context.Background(), db, &note.Note{TenantID: tenantID, ConversationID: convID, Text: "keep this text", Date: &date})
	if err != nil {
		t.Fatal(err)
	}
	svc := note.NewNoteService(repo, db, &config.Config{}, logger.NewLogger("error", "text"))
	newDate := "2026-10-12"
	if _, err := svc.Update(context.Background(), tenantID, id, note.UpdateNoteRequest{Date: &newDate}); err != nil {
		t.Fatal(err)
	}
	newText := "new text"
	if _, err := svc.Update(context.Background(), tenantID, id, note.UpdateNoteRequest{Text: &newText}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(context.Background(), db, tenantID, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != newText || got.Date == nil || got.Date.Format("2006-01-02") != newDate || got.ConversationID != convID {
		t.Fatalf("partial update lost existing fields: %+v", got)
	}
	if _, err := svc.Update(context.Background(), foreign, id, note.UpdateNoteRequest{Text: &newText}); err == nil {
		t.Error("note update accepted foreign tenant")
	}
}

func TestUUIDWebhookUpdatesStayWithinTenant(t *testing.T) {
	db := uuidTestDB(t)
	channelID := insertUUID(t, db, `INSERT INTO channels (name, type) VALUES ('test', 'whatsapp') RETURNING id`)
	tenantIDs := []uuid.UUID{testTenant(t, db), testTenant(t, db)}
	messageIDs := make([]uuid.UUID, len(tenantIDs))
	for i, tenantID := range tenantIDs {
		contactID := testContact(t, db, tenantID)
		profileID := testProfile(t, db, tenantID, contactID, channelID)
		convID := insertUUID(t, db, `INSERT INTO conversations (tenant_id, profile_id, channel_id, status) VALUES ($1,$2,$3,'unassigned') RETURNING id`, tenantID, profileID, channelID)
		messageIDs[i] = insertUUID(t, db, `INSERT INTO messages (tenant_id, conversation_id, sender_id, sender_type, webhook_message_id, status) VALUES ($1,$2,$3,'contact','shared-provider-id','unread') RETURNING id`, tenantID, convID, contactID)
	}
	repo := message.NewMessageRepository()
	if err := repo.UpdateStatusByWebhookID(context.Background(), db, tenantIDs[0], "shared-provider-id", "read"); err != nil {
		t.Fatal(err)
	}
	for i, id := range messageIDs {
		var status string
		want := "unread"
		if i == 0 {
			want = "read"
		}
		if err := db.QueryRow(`SELECT status FROM messages WHERE id=$1`, id).Scan(&status); err != nil || status != want {
			t.Fatalf("message %s status=%s, want=%s, err=%v", id, status, want, err)
		}
	}
	_ = repo.UpdateWebhookID(context.Background(), db, tenantIDs[0], messageIDs[1], "foreign-overwrite")
	var providerID string
	if err := db.QueryRow(`SELECT webhook_message_id FROM messages WHERE id=$1`, messageIDs[1]).Scan(&providerID); err != nil || providerID != "shared-provider-id" {
		t.Fatalf("foreign webhook ID changed: %s, err=%v", providerID, err)
	}
	if err := repo.UpdateWebhookID(context.Background(), db, tenantIDs[0], messageIDs[0], "local-update"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT webhook_message_id FROM messages WHERE id=$1`, messageIDs[0]).Scan(&providerID); err != nil || providerID != "local-update" {
		t.Fatalf("local webhook ID not updated: %s, err=%v", providerID, err)
	}
}
