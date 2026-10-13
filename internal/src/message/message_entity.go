package message

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Reactions             []Reaction      `json:"reactions"`
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
	Reply                 *Quote          `json:"reply,omitempty"`
	IsEdited              bool            `json:"is_edited"`
	EditedAt              *time.Time      `json:"edited_at,omitempty"`
}

type Reaction struct {
	ActorKey   string    `json:"actor_key"`
	SenderType string    `json:"sender_type"`
	Emoji      string    `json:"emoji"`
	ReactedAt  time.Time `json:"reacted_at"`
}

type Quote struct {
	MessageID         *uuid.UUID      `json:"message_id,omitempty"`
	ProviderMessageID string          `json:"provider_message_id"`
	SenderType        string          `json:"sender_type"`
	SenderName        string          `json:"sender_name,omitempty"`
	Text              *string         `json:"text,omitempty"`
	Attachment        json.RawMessage `json:"attachment,omitempty"`
}

// Mutation is a provider event that changes an existing message, not a new bubble.
type Mutation struct {
	Kind             string    `json:"kind"`
	TargetProviderID string    `json:"target_provider_id"`
	ActorKey         string    `json:"actor_key"`
	SenderType       string    `json:"sender_type"`
	Text             *string   `json:"text"`
	Emoji            string    `json:"emoji"`
	EventID          string    `json:"event_id"`
	EventAt          time.Time `json:"event_at"`
}

type Attachment struct {
	IsAnimated bool   `json:"is_animated,omitempty"`
	Type       string `json:"type"`
	URL        string `json:"url,omitempty"`
	Mimetype   string `json:"mimetype,omitempty"`
	FileName   string `json:"fileName,omitempty"`
	Size       int64  `json:"size,omitempty"`
	Caption    string `json:"caption,omitempty"`
	Status     string `json:"status,omitempty"`
}
