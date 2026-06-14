package message

import (
	"encoding/json"
	"time"
)

type Message struct {
	ID                   int              `json:"id"`
	TenantID             int              `json:"tenant_id"`
	Text                 *string          `json:"text,omitempty"`
	Attachment           json.RawMessage  `json:"attachment,omitempty"`
	Status               string           `json:"status"`
	SenderID             int              `json:"sender_id"`
	SenderType           string           `json:"sender_type"`
	WebhookMessageID     *string          `json:"webhook_message_id,omitempty"`
	WebhookMessageReplyID *string         `json:"webhook_message_reply_id,omitempty"`
	ConversationID       int              `json:"conversation_id"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}
