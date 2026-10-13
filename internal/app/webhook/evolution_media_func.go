package webhook

import (
	"encoding/json"
	"strings"
)

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
