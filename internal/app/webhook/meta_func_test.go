package webhook

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/utils/logger"
)

func TestProcessMetaEvent_NotifiesNewMessage(t *testing.T) {
	notifier := &mockNotifier{}

	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "Existing", Facebook: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Type: channelType}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001e"), ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-0000000001f4"), nil
		},
	}

	svc := NewWebhookService(&mockDeviceRepo{}, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, &mockMetaTenantRepo{}, nil, logger.NewLogger("debug", "text"), nil, notifier)

	payload := &MetaWebhookPayload{
		Object: "page",
		Entry: []MetaWebhookEntry{
			{
				ID:   "page_123",
				Time: 1700000000,
				Messaging: []MetaWebhookMessage{
					{
						Sender:  &MetaSender{ID: "fb_sender_1"},
						Message: &MetaMessage{MID: "mid_001", Text: "Hello from facebook"},
					},
				},
			},
		},
	}

	err := svc.ProcessMetaEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ev := notifier.last()
	if ev == nil {
		t.Fatal("expected a websocket notification")
	}
	if ev.tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected tenantID 1, got %s", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	msg, ok := ev.data.(*message.Message)
	if !ok {
		t.Fatalf("expected data *message.Message, got %T", ev.data)
	}
	if msg.ID != uuid.MustParse("00000000-0000-4000-8000-0000000001f4") {
		t.Errorf("expected message id 500, got %s", msg.ID)
	}
	if msg.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000001e") {
		t.Errorf("expected conversation_id 30, got %s", msg.ConversationID)
	}
	if msg.SenderType != "contact" {
		t.Errorf("expected sender_type contact, got %s", msg.SenderType)
	}
	if msg.Text == nil || *msg.Text != "Hello from facebook" {
		t.Errorf("expected text 'Hello from facebook', got %v", msg.Text)
	}
}
