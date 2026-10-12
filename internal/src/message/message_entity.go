package message

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID                    uuid.UUID       `json:"id"`
	TenantID              uuid.UUID       `json:"tenant_id"`
	Text                  *string         `json:"text,omitempty"`
	Attachment            json.RawMessage `json:"attachment,omitempty"`
	Status                string          `json:"status"`
	SenderID              uuid.UUID       `json:"sender_id"`
	SenderType            string          `json:"sender_type"`
	WebhookMessageID      *string         `json:"webhook_message_id,omitempty"`
	WebhookMessageReplyID *string         `json:"webhook_message_reply_id,omitempty"`
	ConversationID        uuid.UUID       `json:"conversation_id"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type Attachment struct {
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	Mimetype string `json:"mimetype,omitempty"`
	FileName string `json:"fileName,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Status   string `json:"status,omitempty"`
}
