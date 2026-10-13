package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"centrachannel/internal/src/message"
)

func extractEvolutionQuote(raw json.RawMessage, contextInfo *EvolutionContextInfo, key EvolutionMessageKey) *message.Quote {
	if contextInfo == nil {
		var content map[string]json.RawMessage
		if json.Unmarshal(raw, &content) != nil {
			return nil
		}
		for _, name := range []string{"ephemeralMessage", "documentWithCaptionMessage", "viewOnceMessage", "viewOnceMessageV2", "viewOnceMessageV2Extension"} {
			if wrapper, ok := content[name]; ok {
				var wrapped struct {
					Message json.RawMessage `json:"message"`
				}
				if json.Unmarshal(wrapper, &wrapped) == nil {
					return extractEvolutionQuote(wrapped.Message, nil, key)
				}
			}
		}
		for _, value := range content {
			var item struct {
				ContextInfo *EvolutionContextInfo `json:"contextInfo"`
			}
			if json.Unmarshal(value, &item) == nil && item.ContextInfo != nil && item.ContextInfo.StanzaID != "" {
				contextInfo = item.ContextInfo
				break
			}
		}
	}
	if contextInfo == nil || contextInfo.StanzaID == "" {
		return nil
	}
	quote := &message.Quote{ProviderMessageID: contextInfo.StanzaID}
	if contextInfo.Participant != "" {
		if contextInfo.Participant == key.RemoteJid || contextInfo.Participant == key.RemoteJidAlt {
			quote.SenderType = "contact"
		} else if strings.HasSuffix(contextInfo.Participant, "@s.whatsapp.net") && strings.HasSuffix(key.RemoteJid, "@s.whatsapp.net") {
			quote.SenderType = "user"
		}
	}
	var quoted EvolutionMessage
	if json.Unmarshal(contextInfo.QuotedMessage, &quoted) != nil {
		return quote
	}
	var contents map[string]json.RawMessage
	_ = json.Unmarshal(contextInfo.QuotedMessage, &contents)
	kind := ""
	for name := range contents {
		if name != "messageContextInfo" {
			kind = name
			break
		}
	}
	quote.Text, quote.Attachment = extractMessageContent(quoted, kind)
	if len(quote.Attachment) > 0 {
		var att map[string]interface{}
		if json.Unmarshal(quote.Attachment, &att) == nil {
			// Quoted provider URLs are encrypted; render a type/filename snapshot.
			delete(att, "url")
			delete(att, "base64")
			quote.Attachment, _ = json.Marshal(att)
		}
	}
	return quote
}

func isEvolutionGroup(jid string) bool {
	return strings.HasSuffix(jid, "@g.us") || jid == "status@broadcast"
}

func (value *EvolutionTimestamp) UnmarshalJSON(raw []byte) error {
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		parsed, err := number.Int64()
		if err != nil {
			return err
		}
		*value = EvolutionTimestamp(parsed)
		return nil
	}
	var parts struct {
		Low  uint32 `json:"low"`
		High int32  `json:"high"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return err
	}
	*value = EvolutionTimestamp(int64(parts.High)<<32 | int64(parts.Low))
	return nil
}

func evolutionEventTime(value EvolutionTimestamp) time.Time {
	if value == 0 {
		return time.Now().UTC()
	}
	if value > 100_000_000_000 {
		return time.UnixMilli(int64(value)).UTC()
	}
	return time.Unix(int64(value), 0).UTC()
}

func unwrapEvolutionMessage(content EvolutionMessage) EvolutionMessage {
	for {
		var inner *EvolutionMessage
		switch {
		case content.EphemeralMessage != nil:
			inner = content.EphemeralMessage.Message
		case content.DocumentWithCaptionMessage != nil:
			inner = content.DocumentWithCaptionMessage.Message
		case content.ViewOnceMessage != nil:
			inner = content.ViewOnceMessage.Message
		case content.ViewOnceMessageV2 != nil:
			inner = content.ViewOnceMessageV2.Message
		case content.ViewOnceMessageV2Extension != nil:
			inner = content.ViewOnceMessageV2Extension.Message
		}
		if inner == nil {
			return content
		}
		previous := content.Base64
		content = *inner
		if previous != "" {
			content.Base64 = previous
		}
	}
}

func extractMessageContent(evtMsg EvolutionMessage, msgType string) (*string, json.RawMessage) {
	evtMsg = unwrapEvolutionMessage(evtMsg)
	if evtMsg.DocumentWithCaptionMessage != nil && evtMsg.DocumentWithCaptionMessage.Message != nil {
		inner := *evtMsg.DocumentWithCaptionMessage.Message
		if evtMsg.Base64 != "" {
			inner.Base64 = evtMsg.Base64
		}
		return extractMessageContent(inner, msgType)
	}
	if evtMsg.EphemeralMessage != nil && evtMsg.EphemeralMessage.Message != nil {
		inner := *evtMsg.EphemeralMessage.Message
		if evtMsg.Base64 != "" {
			inner.Base64 = evtMsg.Base64
		}
		return extractMessageContent(inner, msgType)
	}
	if evtMsg.Base64 != "" {
		if evtMsg.StickerMessage != nil {
			evtMsg.StickerMessage.Base64 = evtMsg.Base64
		}
		if evtMsg.ImageMessage != nil {
			evtMsg.ImageMessage.Base64 = evtMsg.Base64
		}
		if evtMsg.VideoMessage != nil {
			evtMsg.VideoMessage.Base64 = evtMsg.Base64
		}
		if evtMsg.AudioMessage != nil {
			evtMsg.AudioMessage.Base64 = evtMsg.Base64
		}
		if evtMsg.DocumentMessage != nil {
			evtMsg.DocumentMessage.Base64 = evtMsg.Base64
		}
	}
	if evtMsg.Conversation != nil && *evtMsg.Conversation != "" {
		return evtMsg.Conversation, nil
	}
	if evtMsg.ExtendedTextMessage != nil && evtMsg.ExtendedTextMessage.Text != "" {
		return &evtMsg.ExtendedTextMessage.Text, nil
	}

	if evtMsg.ImageMessage != nil {
		att := map[string]interface{}{
			"type":     "image",
			"mimetype": evtMsg.ImageMessage.Mimetype,
		}
		if evtMsg.ImageMessage.Caption != "" {
			att["caption"] = evtMsg.ImageMessage.Caption
		}
		if evtMsg.ImageMessage.URL != "" && strings.HasPrefix(evtMsg.ImageMessage.URL, "http") {
			att["url"] = evtMsg.ImageMessage.URL
		}
		if evtMsg.ImageMessage.Base64 != "" {
			att["base64"] = evtMsg.ImageMessage.Base64
		}
		attJSON, _ := json.Marshal(att)
		caption := evtMsg.ImageMessage.Caption
		if caption == "" {
			return nil, attJSON
		}
		return &caption, attJSON
	}

	if evtMsg.VideoMessage != nil {
		att := map[string]interface{}{
			"type":     "video",
			"mimetype": evtMsg.VideoMessage.Mimetype,
		}
		if evtMsg.VideoMessage.Caption != "" {
			att["caption"] = evtMsg.VideoMessage.Caption
		}
		if evtMsg.VideoMessage.URL != "" && strings.HasPrefix(evtMsg.VideoMessage.URL, "http") {
			att["url"] = evtMsg.VideoMessage.URL
		}
		if evtMsg.VideoMessage.Base64 != "" {
			att["base64"] = evtMsg.VideoMessage.Base64
		}
		attJSON, _ := json.Marshal(att)
		caption := evtMsg.VideoMessage.Caption
		if caption == "" {
			return nil, attJSON
		}
		return &caption, attJSON
	}

	if evtMsg.AudioMessage != nil {
		att := map[string]interface{}{
			"type":     "audio",
			"mimetype": evtMsg.AudioMessage.Mimetype,
		}
		if evtMsg.AudioMessage.URL != "" && strings.HasPrefix(evtMsg.AudioMessage.URL, "http") {
			att["url"] = evtMsg.AudioMessage.URL
		}
		if evtMsg.AudioMessage.Base64 != "" {
			att["base64"] = evtMsg.AudioMessage.Base64
		}
		attJSON, _ := json.Marshal(att)
		return nil, attJSON
	}

	if evtMsg.DocumentMessage != nil {
		att := map[string]interface{}{
			"type":     "document",
			"mimetype": evtMsg.DocumentMessage.Mimetype,
			"fileName": evtMsg.DocumentMessage.FileName,
		}
		if evtMsg.DocumentMessage.URL != "" && strings.HasPrefix(evtMsg.DocumentMessage.URL, "http") {
			att["url"] = evtMsg.DocumentMessage.URL
		}
		if evtMsg.DocumentMessage.Base64 != "" {
			att["base64"] = evtMsg.DocumentMessage.Base64
		}
		if evtMsg.DocumentMessage.Caption != "" {
			att["caption"] = evtMsg.DocumentMessage.Caption
		}
		attJSON, _ := json.Marshal(att)
		if evtMsg.DocumentMessage.Caption != "" {
			caption := evtMsg.DocumentMessage.Caption
			return &caption, attJSON
		}
		return nil, attJSON
	}

	if sticker := evtMsg.StickerMessage; sticker != nil {
		att := map[string]interface{}{
			"type": "sticker", "mimetype": sticker.Mimetype, "is_animated": sticker.IsAnimated,
		}
		if sticker.URL != "" && strings.HasPrefix(sticker.URL, "http") {
			att["url"] = sticker.URL
		}
		if sticker.Base64 != "" {
			att["base64"] = sticker.Base64
		}
		raw, _ := json.Marshal(att)
		return nil, raw
	}
	unsupported := map[string]string{"type": "unsupported", "message_type": msgType}
	if msgType == "" {
		return nil, nil
	}
	attJSON, _ := json.Marshal(unsupported)
	return nil, attJSON
}

func extractEvolutionMutation(key EvolutionMessageKey, content EvolutionMessage, timestamp EvolutionTimestamp) (message.Mutation, bool, error) {
	content = unwrapEvolutionMessage(content)
	event := message.Mutation{EventID: key.ID, EventAt: evolutionEventTime(timestamp), ActorKey: "contact", SenderType: "contact"}
	if key.FromMe {
		event.ActorKey = "user"
		event.SenderType = "user"
	}
	switch {
	case content.ReactionMessage != nil:
		reaction := content.ReactionMessage
		event.Kind = "reaction"
		event.TargetProviderID = reaction.Key.ID
		event.Emoji = reaction.Text
		if reaction.SenderTimestampMS != 0 {
			event.EventAt = evolutionEventTime(reaction.SenderTimestampMS)
		}
	case content.EditedMessage != nil && content.EditedMessage.Message != nil:
		event.Kind = "edit"
		event.TargetProviderID = key.ID
		edited := unwrapEvolutionMessage(*content.EditedMessage.Message)
		event.Text, _ = extractMessageContent(edited, "")
		if edited.Conversation == nil && edited.ExtendedTextMessage == nil && edited.ImageMessage == nil && edited.VideoMessage == nil && edited.DocumentMessage == nil {
			return event, true, fmt.Errorf("unsupported edited content")
		}
		encoded, _ := json.Marshal(content.EditedMessage)
		digest := sha256.Sum256(encoded)
		event.EventID = key.ID + ":" + event.EventAt.Format(time.RFC3339Nano) + ":" + hex.EncodeToString(digest[:])
	case content.ProtocolMessage != nil && content.ProtocolMessage.EditedMessage != nil:
		protocol := content.ProtocolMessage
		inner := EvolutionMessage{EditedMessage: &EvolutionMessageContainer{Message: protocol.EditedMessage}}
		if protocol.Key.ID != "" {
			key.ID = protocol.Key.ID
		}
		if protocol.TimestampMS != 0 {
			timestamp = protocol.TimestampMS
		}
		return extractEvolutionMutation(key, inner, timestamp)
	case content.SecretEncryptedMessage != nil:
		return event, true, fmt.Errorf("encrypted edit requires the Evolution secret-edit adapter")
	default:
		return event, false, nil
	}
	if event.TargetProviderID == "" {
		return event, true, fmt.Errorf("message mutation has no target")
	}
	return event, true, nil
}
