package webhook

import "encoding/json"

type MetaWebhookPayload struct {
	Object string             `json:"object"`
	Entry  []MetaWebhookEntry `json:"entry"`
}

type MetaWebhookEntry struct {
	ID        string               `json:"id"`
	Time      int64                `json:"time"`
	Messaging []MetaWebhookMessage `json:"messaging"`
}

type MetaWebhookMessage struct {
	Sender    *MetaSender    `json:"sender,omitempty"`
	Recipient *MetaRecipient `json:"recipient,omitempty"`
	Timestamp int64          `json:"timestamp"`
	Message   *MetaMessage   `json:"message,omitempty"`
}

type MetaSender struct {
	ID string `json:"id"`
}

type MetaRecipient struct {
	ID string `json:"id"`
}

type MetaMessage struct {
	MID  string `json:"mid"`
	Text string `json:"text,omitempty"`
}

type EvolutionWebhookPayload struct {
	Event    string          `json:"event"`
	Instance string          `json:"instance"`
	Data     json.RawMessage `json:"data,omitempty"`
	APIKey   string          `json:"apikey,omitempty"`
}

type EvolutionMessageUpsert struct {
	RawMessage       json.RawMessage       `json:"-"`
	Key              EvolutionMessageKey   `json:"key"`
	PushName         string                `json:"pushName"`
	Message          EvolutionMessage      `json:"message"`
	MessageType      string                `json:"messageType"`
	ContextInfo      *EvolutionContextInfo `json:"contextInfo,omitempty"`
	MessageTimestamp EvolutionTimestamp    `json:"messageTimestamp,omitempty"`
}

type EvolutionConnectionUpdate struct {
	Instance     string `json:"instance"`
	State        string `json:"state"`
	StatusReason int    `json:"statusReason,omitempty"`
}

type EvolutionMessageUpdate struct {
	KeyID            string             `json:"keyId"`
	Status           string             `json:"status"`
	RemoteJid        string             `json:"remoteJid,omitempty"`
	FromMe           bool               `json:"fromMe,omitempty"`
	Message          *EvolutionMessage  `json:"message,omitempty"`
	MessageTimestamp EvolutionTimestamp `json:"messageTimestamp,omitempty"`
	TimestampMs      EvolutionTimestamp `json:"timestampMs,omitempty"`
}

type EvolutionMessageKey struct {
	RemoteJid    string `json:"remoteJid"`
	FromMe       bool   `json:"fromMe"`
	ID           string `json:"id"`
	RemoteJidAlt string `json:"remoteJidAlt,omitempty"`
}

type EvolutionContextInfo struct {
	StanzaID      string          `json:"stanzaId"`
	Participant   string          `json:"participant"`
	QuotedMessage json.RawMessage `json:"quotedMessage"`
}

type EvolutionMessage struct {
	ReactionMessage            *EvolutionReactionMessage  `json:"reactionMessage,omitempty"`
	EditedMessage              *EvolutionMessageContainer `json:"editedMessage,omitempty"`
	ProtocolMessage            *EvolutionProtocolMessage  `json:"protocolMessage,omitempty"`
	SecretEncryptedMessage     *EvolutionSecretMessage    `json:"secretEncryptedMessage,omitempty"`
	ViewOnceMessage            *EvolutionMessageContainer `json:"viewOnceMessage,omitempty"`
	ViewOnceMessageV2          *EvolutionMessageContainer `json:"viewOnceMessageV2,omitempty"`
	ViewOnceMessageV2Extension *EvolutionMessageContainer `json:"viewOnceMessageV2Extension,omitempty"`
	StickerMessage             *EvolutionStickerMessage   `json:"stickerMessage,omitempty"`
	DocumentWithCaptionMessage *struct {
		Message *EvolutionMessage `json:"message"`
	} `json:"documentWithCaptionMessage,omitempty"`
	EphemeralMessage *struct {
		Message *EvolutionMessage `json:"message"`
	} `json:"ephemeralMessage,omitempty"`
	Base64       string  `json:"base64,omitempty"`
	Conversation *string `json:"conversation,omitempty"`

	ExtendedTextMessage *struct {
		Text string `json:"text"`
	} `json:"extendedTextMessage,omitempty"`

	ImageMessage *struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	} `json:"imageMessage,omitempty"`

	VideoMessage *struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	} `json:"videoMessage,omitempty"`

	AudioMessage *struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Base64   string `json:"base64,omitempty"`
	} `json:"audioMessage,omitempty"`

	DocumentMessage *struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		FileName string `json:"fileName,omitempty"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	} `json:"documentMessage,omitempty"`
}

type EvolutionTimestamp int64

type EvolutionMessageContainer struct {
	Message *EvolutionMessage `json:"message"`
}
type EvolutionReactionMessage struct {
	Key               EvolutionMessageKey `json:"key"`
	Text              string              `json:"text"`
	SenderTimestampMS EvolutionTimestamp  `json:"senderTimestampMs"`
}
type EvolutionProtocolMessage struct {
	Key           EvolutionMessageKey `json:"key"`
	Type          json.RawMessage     `json:"type"`
	EditedMessage *EvolutionMessage   `json:"editedMessage"`
	TimestampMS   EvolutionTimestamp  `json:"timestampMs"`
}
type EvolutionSecretMessage struct {
	TargetMessageKey EvolutionMessageKey `json:"targetMessageKey"`
	SecretEncType    int                 `json:"secretEncType"`
}

type EvolutionStickerMessage struct {
	URL        string `json:"url"`
	Mimetype   string `json:"mimetype"`
	Base64     string `json:"base64,omitempty"`
	IsAnimated bool   `json:"isAnimated"`
}
