package webhook

import (
	"context"
	"testing"

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
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: 10, TenantID: 1, FirstName: "Existing", Facebook: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
			return &profile.Profile{ID: 20, ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: 1, Type: channelType}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, profileID int, channelID int) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: 30, ProfileID: 20, Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) { return 500, nil },
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
	if ev.tenantID != 1 {
		t.Errorf("expected tenantID 1, got %d", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	msg, ok := ev.data.(*message.Message)
	if !ok {
		t.Fatalf("expected data *message.Message, got %T", ev.data)
	}
	if msg.ID != 500 {
		t.Errorf("expected message id 500, got %d", msg.ID)
	}
	if msg.ConversationID != 30 {
		t.Errorf("expected conversation_id 30, got %d", msg.ConversationID)
	}
	if msg.SenderType != "contact" {
		t.Errorf("expected sender_type contact, got %s", msg.SenderType)
	}
	if msg.Text == nil || *msg.Text != "Hello from facebook" {
		t.Errorf("expected text 'Hello from facebook', got %v", msg.Text)
	}
}
