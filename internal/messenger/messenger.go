package messenger

import "encoding/json"

type OutgoingMessage struct {
	ChannelType string
	RecipientID string
	Text        *string
	Attachment  json.RawMessage
	Quoted      *QuotedMessage
}

type Messenger interface {
	Send(msg *OutgoingMessage) (string, error)
}

// QuotedMessage carries provider-independent content for an outgoing reply.
type QuotedMessage struct {
	ProviderMessageID string
	FromMe            bool
	Text              *string
	Attachment        json.RawMessage
}
