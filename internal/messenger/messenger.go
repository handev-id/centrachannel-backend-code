package messenger

import "encoding/json"

type OutgoingMessage struct {
	ChannelType string
	RecipientID string
	Text        *string
	Attachment  json.RawMessage
}

type Messenger interface {
	Send(msg *OutgoingMessage) (string, error)
}
