package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/whatsapp_device"
	"github.com/google/uuid"
)

func newContentTestService(t *testing.T, capture func(*message.Message)) *webhookService {
	t.Helper()
	tenantID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	svc := newTestService(
		&mockDeviceRepo{getByWhatsappIDFunc: func(context.Context, whatsapp_device.DBTX, string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{TenantID: tenantID}, nil
		}},
		&mockContactRepo{getByPhoneFunc: func(context.Context, contact.DBTX, uuid.UUID, string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.New(), TenantID: tenantID, FirstName: "Customer"}, nil
		}},
		&mockProfileRepo{getByExternalIDAndChannelIDFunc: func(context.Context, profile.DBTX, uuid.UUID, string, uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002")}, nil
		}},
		&mockChannelRepo{getByTypeFunc: func(context.Context, channel.DBTX, string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000003"), Type: "whatsapp"}, nil
		}},
		&mockConvRepo{findOpenByProfileAndChannelFunc: func(context.Context, conversation.DBTX, uuid.UUID, uuid.UUID, uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-000000000004")}, nil
		}},
		&mockMsgRepo{createFunc: func(_ context.Context, _ message.DBTX, m *message.Message) (uuid.UUID, error) {
			capture(m)
			return uuid.New(), nil
		}},
	)
	return svc.(*webhookService)
}

func TestIncomingQuoteKeepsOriginalContext(t *testing.T) {
	var created *message.Message
	svc := newContentTestService(t, func(m *message.Message) { created = m })
	raw := json.RawMessage(`{"key":{"id":"reply-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"messageType":"extendedTextMessage","message":{"extendedTextMessage":{"text":"Reply","contextInfo":{"stanzaId":"original-id","participant":"628123456789@s.whatsapp.net","quotedMessage":{"conversation":"Original"}}}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.upsert", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if created == nil {
		t.Fatal("no reply message created")
	}
	wire, _ := json.Marshal(created)
	var response struct {
		Reply *struct {
			ProviderMessageID string `json:"provider_message_id"`
			Text              string `json:"text"`
		} `json:"reply"`
	}
	if err := json.Unmarshal(wire, &response); err != nil {
		t.Fatal(err)
	}
	if response.Reply == nil || response.Reply.ProviderMessageID != "original-id" || response.Reply.Text != "Original" {
		t.Fatalf("quote context lost: %s", wire)
	}
}

func TestReactionUpsertDoesNotCreateBubble(t *testing.T) {
	created := 0
	svc := newContentTestService(t, func(*message.Message) { created++ })
	var mutation *message.Mutation
	svc.msgRepo.(*mockMsgRepo).applyMutationFunc = func(_ context.Context, _ *sql.DB, _, _, _ uuid.UUID, event message.Mutation) (*message.Message, error) {
		mutation = &event
		return &message.Message{ID: uuid.New()}, nil
	}
	raw := json.RawMessage(`{"key":{"id":"reaction-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"messageType":"reactionMessage","message":{"reactionMessage":{"key":{"id":"target-id","fromMe":true},"text":"👍","senderTimestampMs":"1791894000123"}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.upsert", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if created != 0 || mutation == nil || mutation.TargetProviderID != "target-id" || mutation.Emoji != "👍" {
		t.Fatalf("reaction became a bubble: created=%d mutation=%+v", created, mutation)
	}
}

func TestEditedUpdateDoesNotBecomeDeliveryStatus(t *testing.T) {
	svc := newContentTestService(t, func(*message.Message) { t.Fatal("edit created a bubble") })
	var mutation *message.Mutation
	statusUpdates := 0
	repo := svc.msgRepo.(*mockMsgRepo)
	repo.applyMutationFunc = func(_ context.Context, _ *sql.DB, _, _, _ uuid.UUID, event message.Mutation) (*message.Message, error) {
		mutation = &event
		return &message.Message{ID: uuid.New()}, nil
	}
	repo.updateStatusByWebhookIDFunc = func(context.Context, message.DBTX, uuid.UUID, string, string) error { statusUpdates++; return nil }
	raw := json.RawMessage(`{"keyId":"target-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false,"status":"SERVER_ACK","messageTimestamp":1791894001,"message":{"editedMessage":{"message":{"conversation":"Edited"}}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if statusUpdates != 0 || mutation == nil || mutation.Text == nil || *mutation.Text != "Edited" {
		t.Fatalf("edit routed as status: statuses=%d mutation=%+v", statusUpdates, mutation)
	}
}

func TestLegacyEditedWebhookUpdatesTarget(t *testing.T) {
	svc := newContentTestService(t, func(*message.Message) { t.Fatal("edit created a bubble") })
	var mutation *message.Mutation
	svc.msgRepo.(*mockMsgRepo).applyMutationFunc = func(_ context.Context, _ *sql.DB, _, _, _ uuid.UUID, event message.Mutation) (*message.Message, error) {
		mutation = &event
		return &message.Message{ID: uuid.New()}, nil
	}
	raw := json.RawMessage(`{"key":{"id":"target-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"type":14,"editedMessage":{"conversation":"Legacy edit"},"timestampMs":"1791894000123"}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.edited", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if mutation == nil || mutation.Text == nil || *mutation.Text != "Legacy edit" {
		t.Fatalf("legacy edit lost: %+v", mutation)
	}
}

func TestIncomingQuoteResolvesStoredContactAndMedia(t *testing.T) {
	var created *message.Message
	svc := newContentTestService(t, func(m *message.Message) { created = m })
	targetID := uuid.New()
	svc.msgRepo.(*mockMsgRepo).getByProviderIDFunc = func(_ context.Context, _ message.DBTX, tenant, profileID, channelID uuid.UUID, id string) (*message.Message, error) {
		if tenant != uuid.MustParse("00000000-0000-4000-8000-000000000001") || profileID != uuid.MustParse("00000000-0000-4000-8000-000000000002") || channelID != uuid.MustParse("00000000-0000-4000-8000-000000000003") || id != "original-id" {
			t.Fatal("quote lookup is not scoped to the peer")
		}
		return &message.Message{ID: targetID, SenderType: "contact", Attachment: json.RawMessage(`{"type":"image","url":"https://storage.example/photo.webp","status":"ready","caption":"Current caption"}`)}, nil
	}
	raw := json.RawMessage(`{"key":{"id":"reply-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"messageType":"extendedTextMessage","message":{"extendedTextMessage":{"text":"Reply","contextInfo":{"stanzaId":"original-id","participant":"123456789@lid","quotedMessage":{"imageMessage":{"url":"https://mmg.whatsapp.net/encrypted.enc","caption":"Snapshot caption","mimetype":"image/webp"}}}}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.upsert", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if created == nil || created.Reply == nil || created.Reply.SenderType != "contact" || created.Reply.MessageID == nil || *created.Reply.MessageID != targetID || created.Reply.SenderName != "Customer" {
		t.Fatalf("LID quote author not resolved: %+v", created)
	}
	var att message.Attachment
	if err := json.Unmarshal(created.Reply.Attachment, &att); err != nil {
		t.Fatal(err)
	}
	if att.URL != "https://storage.example/photo.webp" || att.Caption != "Snapshot caption" {
		t.Fatalf("quote lost stored thumbnail or snapshot caption: %s", created.Reply.Attachment)
	}
}
