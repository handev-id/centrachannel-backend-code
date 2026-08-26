package whatsapp_device

import (
	"context"
	"fmt"
	"time"

	"centrachannel/internal/utils/logger"
)

type WhatsAppClient interface {
	SendMessage(ctx context.Context, device *WhatsAppDevice, to string, text string) (*MessageResult, error)
	GetQR(ctx context.Context, device *WhatsAppDevice) (string, error)
	GetPairingCode(ctx context.Context, device *WhatsAppDevice, phoneNumber string) (string, error)
	CheckConnection(ctx context.Context, device *WhatsAppDevice) (bool, error)
	Disconnect(ctx context.Context, device *WhatsAppDevice) error
	CreateInstance(ctx context.Context, device *WhatsAppDevice) error
	DeleteInstance(ctx context.Context, device *WhatsAppDevice) error
	SetWebhook(ctx context.Context, device *WhatsAppDevice, webhookURL string) error
	MarkMessagesAsRead(ctx context.Context, device *WhatsAppDevice, messages []ReadMessageKey) error
}

type ReadMessageKey struct {
	ID        string `json:"id"`
	FromMe    bool   `json:"fromMe"`
	RemoteJid string `json:"remoteJid"`
}

type MessageResult struct {
	MessageID string    `json:"message_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type DeviceConfig struct {
	WebhookURL string `json:"webhook_url,omitempty"`
	AutoReply  bool   `json:"auto_reply,omitempty"`
	ReplyText  string `json:"reply_text,omitempty"`
}

type mockClient struct {
	logger *logger.Logger
}

func NewMockClient(logger *logger.Logger) WhatsAppClient {
	return &mockClient{logger: logger}
}

func (c *mockClient) SendMessage(ctx context.Context, device *WhatsAppDevice, to string, text string) (*MessageResult, error) {
	c.logger.Info("Mock WhatsApp send: device_id=%d, to=%s", device.ID, to)
	return &MessageResult{
		MessageID: fmt.Sprintf("mock_msg_%d", time.Now().UnixNano()),
		Status:    "sent",
		Timestamp: time.Now(),
	}, nil
}

func (c *mockClient) GetQR(ctx context.Context, device *WhatsAppDevice) (string, error) {
	c.logger.Info("Mock WhatsApp QR generation: device_id=%d", device.ID)
	return "mock_qr_data_for_device_" + fmt.Sprint(device.ID), nil
}

func (c *mockClient) GetPairingCode(ctx context.Context, device *WhatsAppDevice, phoneNumber string) (string, error) {
	c.logger.Info("Mock WhatsApp pairing code: device_id=%d, phone=%s", device.ID, phoneNumber)
	return "ABCD-1234", nil
}

func (c *mockClient) CheckConnection(ctx context.Context, device *WhatsAppDevice) (bool, error) {
	c.logger.Info("Mock WhatsApp connection check: device_id=%d", device.ID)
	return true, nil
}

func (c *mockClient) Disconnect(ctx context.Context, device *WhatsAppDevice) error {
	c.logger.Info("Mock WhatsApp disconnect: device_id=%d", device.ID)
	return nil
}

func (c *mockClient) CreateInstance(ctx context.Context, device *WhatsAppDevice) error {
	c.logger.Info("Mock WhatsApp create instance: device_id=%d, name=%s", device.ID, device.WhatsappID)
	return nil
}

func (c *mockClient) DeleteInstance(ctx context.Context, device *WhatsAppDevice) error {
	c.logger.Info("Mock WhatsApp delete instance: device_id=%d, name=%s", device.ID, device.WhatsappID)
	return nil
}

func (c *mockClient) SetWebhook(ctx context.Context, device *WhatsAppDevice, webhookURL string) error {
	c.logger.Info("Mock WhatsApp set webhook: device_id=%d, url=%s", device.ID, webhookURL)
	return nil
}

func (c *mockClient) MarkMessagesAsRead(ctx context.Context, device *WhatsAppDevice, messages []ReadMessageKey) error {
	c.logger.Info("Mock WhatsApp mark messages as read: device_id=%d, count=%d", device.ID, len(messages))
	return nil
}
