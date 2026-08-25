package webhook

import "encoding/json"

type MetaWebhookPayload struct {
	Object string             `json:"object"`
	Entry  []MetaWebhookEntry `json:"entry"`
}

type MetaWebhookEntry struct {
	ID        string              `json:"id"`
	Time      int64               `json:"time"`
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
}

type EvolutionMessageUpsert struct {
	Key         EvolutionMessageKey `json:"key"`
	PushName    string              `json:"pushName"`
	Message     EvolutionMessage    `json:"message"`
	MessageType string              `json:"messageType"`
}

type EvolutionConnectionUpdate struct {
	Instance    string `json:"instance"`
	State       string `json:"state"`
	StatusReason int   `json:"statusReason,omitempty"`
}

type EvolutionMessageUpdate struct {
	Key    EvolutionMessageKey   `json:"key"`
	Update EvolutionStatusUpdate `json:"update"`
}

type EvolutionStatusUpdate struct {
	Status string `json:"status"`
}

type EvolutionMessageKey struct {
	RemoteJid string `json:"remoteJid"`
	FromMe    bool   `json:"fromMe"`
	ID        string `json:"id"`
}

type EvolutionMessage struct {
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
		Base64   string `json:"base64,omitempty"`
	} `json:"documentMessage,omitempty"`
}
