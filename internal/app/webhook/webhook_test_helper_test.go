package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/logger"
)

// ---- mock repositories ----

type mockDeviceRepo struct {
	getByWhatsappIDFunc func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error)
	updateFunc          func(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID, device *whatsapp_device.WhatsAppDevice) error
}

func (m *mockDeviceRepo) List(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID) ([]whatsapp_device.WhatsAppDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) GetByID(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID) (*whatsapp_device.WhatsAppDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) GetByWhatsappID(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
	if m.getByWhatsappIDFunc != nil {
		return m.getByWhatsappIDFunc(ctx, q, whatsappID)
	}
	return nil, nil
}
func (m *mockDeviceRepo) Create(ctx context.Context, q whatsapp_device.DBTX, device *whatsapp_device.WhatsAppDevice) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockDeviceRepo) Update(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID, device *whatsapp_device.WhatsAppDevice) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, q, tenantID, id, device)
	}
	return nil
}
func (m *mockDeviceRepo) Delete(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}

type mockContactRepo struct {
	contact.ContactRepository
	getByPhoneFunc func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error)
	createFunc     func(ctx context.Context, q contact.DBTX, c *contact.Contact) (uuid.UUID, error)
}

func (m *mockContactRepo) List(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, limit, offset int, f contact.ListContactQuery) ([]*contact.Contact, int, error) {
	return nil, 0, nil
}
func (m *mockContactRepo) GetByID(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, id uuid.UUID) (*contact.Contact, error) {
	return nil, nil
}
func (m *mockContactRepo) GetByPhone(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
	if m.getByPhoneFunc != nil {
		return m.getByPhoneFunc(ctx, q, tenantID, phone)
	}
	return nil, nil
}
func (m *mockContactRepo) Create(ctx context.Context, q contact.DBTX, c *contact.Contact) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, c)
	}
	return uuid.Nil, nil
}
func (m *mockContactRepo) Update(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, id uuid.UUID, c *contact.Contact) error {
	return nil
}
func (m *mockContactRepo) SoftDelete(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}

type mockProfileRepo struct {
	getByExternalIDAndChannelIDFunc func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error)
	createFunc                      func(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error)
}

func (m *mockProfileRepo) List(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID, channelID uuid.UUID) ([]profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID) (*profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByExternalIDAndChannelID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
	if m.getByExternalIDAndChannelIDFunc != nil {
		return m.getByExternalIDAndChannelIDFunc(ctx, q, tenantID, externalID, channelID)
	}
	return nil, nil
}
func (m *mockProfileRepo) Create(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, p)
	}
	return uuid.Nil, nil
}
func (m *mockProfileRepo) Update(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID, p *profile.Profile) error {
	return nil
}
func (m *mockProfileRepo) GetByContactID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID uuid.UUID) ([]profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByContactIDs(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID][]profile.Profile, error) {
	return map[uuid.UUID][]profile.Profile{}, nil
}

type mockChannelRepo struct {
	getByTypeFunc func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error)
}

func (m *mockChannelRepo) GetByID(ctx context.Context, q channel.DBTX, id uuid.UUID) (*channel.Channel, error) {
	return &channel.Channel{}, nil
}
func (m *mockChannelRepo) GetByType(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
	if m.getByTypeFunc != nil {
		return m.getByTypeFunc(ctx, q, channelType)
	}
	return &channel.Channel{}, nil
}
func (m *mockChannelRepo) List(ctx context.Context, q channel.DBTX) ([]channel.Channel, error) {
	return nil, nil
}

type mockConvRepo struct {
	createFunc                      func(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (uuid.UUID, error)
	updateLastMessageFunc           func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error
	findOpenByProfileAndChannelFunc func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error)
}

func (m *mockConvRepo) GetByID(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) (*conversation.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) Create(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, conv)
	}
	return uuid.Nil, nil
}
func (m *mockConvRepo) UpdateStatus(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	return nil
}
func (m *mockConvRepo) Assign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, agentID uuid.UUID) error {
	return nil
}
func (m *mockConvRepo) Unassign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}
func (m *mockConvRepo) GetTotalUnread(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockConvRepo) MarkRead(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}
func (m *mockConvRepo) ListCursor(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, limit int, status string, channelID, agentID uuid.UUID, search string, lastActivity *time.Time, lastID uuid.UUID) ([]*conversation.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) UpdateLastMessage(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
	if m.updateLastMessageFunc != nil {
		return m.updateLastMessageFunc(ctx, q, tenantID, id, lastMessageJSON, lastAgentID)
	}
	return nil
}
func (m *mockConvRepo) FindOpenByProfileAndChannel(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
	if m.findOpenByProfileAndChannelFunc != nil {
		return m.findOpenByProfileAndChannelFunc(ctx, q, tenantID, profileID, channelID)
	}
	return nil, fmt.Errorf("not found")
}

type mockMsgRepo struct {
	createFunc                  func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error)
	updateStatusByWebhookIDFunc func(ctx context.Context, q message.DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error
}

func (m *mockMsgRepo) Create(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, msg)
	}
	return uuid.Nil, nil
}
func (m *mockMsgRepo) UpdateStatus(ctx context.Context, q message.DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	return nil
}
func (m *mockMsgRepo) UpdateStatusByWebhookID(ctx context.Context, q message.DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
	if m.updateStatusByWebhookIDFunc != nil {
		return m.updateStatusByWebhookIDFunc(ctx, q, tenantID, webhookMessageID, status)
	}
	return nil
}
func (m *mockMsgRepo) ListCursor(ctx context.Context, q message.DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*message.Message, error) {
	return nil, nil
}
func (m *mockMsgRepo) UpdateWebhookID(ctx context.Context, q message.DBTX, tenantID uuid.UUID, id uuid.UUID, webhookMessageID string) error {
	return nil
}
func (m *mockMsgRepo) MarkReadByConversation(ctx context.Context, q message.DBTX, tenantID uuid.UUID, conversationID uuid.UUID) error {
	return nil
}
func (m *mockMsgRepo) GetUnreadByConversation(ctx context.Context, q message.DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]*message.Message, error) {
	return nil, nil
}

// ---- test helpers ----

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

type mockMetaTenantRepo struct{}

func (m *mockMetaTenantRepo) Create(ctx context.Context, q tenant.DBTX, t *tenant.Tenant) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockMetaTenantRepo) CreateRole(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockMetaTenantRepo) CreateUser(ctx context.Context, q tenant.DBTX, u *tenant.User) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockMetaTenantRepo) AttachRole(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error {
	return nil
}
func (m *mockMetaTenantRepo) List(ctx context.Context, q tenant.DBTX) ([]tenant.Tenant, error) {
	return nil, nil
}
func (m *mockMetaTenantRepo) GetByID(ctx context.Context, q tenant.DBTX, id uuid.UUID) (*tenant.Tenant, error) {
	return nil, nil
}
func (m *mockMetaTenantRepo) GetByMetaPageID(ctx context.Context, q tenant.DBTX, pageID string) (*tenant.Tenant, error) {
	return &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Test Tenant"}, nil
}
func (m *mockMetaTenantRepo) GetByMetaInstagramBusinessID(ctx context.Context, q tenant.DBTX, igID string) (*tenant.Tenant, error) {
	return &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Test Tenant"}, nil
}
func (m *mockMetaTenantRepo) Update(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) error {
	return nil
}

func newTestService(deviceRepo whatsapp_device.WhatsAppDeviceRepository, contactRepo contact.ContactRepository, profileRepo profile.ProfileRepository, channelRepo channel.ChannelRepository, convRepo conversation.ConversationRepository, msgRepo message.MessageRepository) WebhookService {
	return NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, &mockMetaTenantRepo{}, nil, logger.NewLogger("debug", "text"), nil)
}

func (m *mockMetaTenantRepo) GetDomains(ctx context.Context, q tenant.DBTX, id uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage("[]"), nil
}
func (m *mockMetaTenantRepo) GetActiveIDByDomain(ctx context.Context, q tenant.DBTX, name string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
