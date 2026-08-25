package webhook

import (
	"encoding/json"
	"fmt"
	"strings"
)

func extractMessageContent(evtMsg EvolutionMessage, msgType string) (*string, json.RawMessage) {
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
			if att["url"] == nil {
				att["url"] = fmt.Sprintf("data:%s;base64,%s", evtMsg.ImageMessage.Mimetype, evtMsg.ImageMessage.Base64)
			}
		}
		if evtMsg.ImageMessage.Base64 == "" && evtMsg.ImageMessage.URL == "" {
			return nil, nil
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
			if att["url"] == nil {
				att["url"] = fmt.Sprintf("data:%s;base64,%s", evtMsg.VideoMessage.Mimetype, evtMsg.VideoMessage.Base64)
			}
		}
		if evtMsg.VideoMessage.Base64 == "" && evtMsg.VideoMessage.URL == "" {
			return nil, nil
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
			if att["url"] == nil {
				att["url"] = fmt.Sprintf("data:%s;base64,%s", evtMsg.AudioMessage.Mimetype, evtMsg.AudioMessage.Base64)
			}
		}
		if evtMsg.AudioMessage.Base64 == "" && evtMsg.AudioMessage.URL == "" {
			return nil, nil
		}
		attJSON, _ := json.Marshal(att)
		return nil, attJSON
	}

	if evtMsg.DocumentMessage != nil {
		att := map[string]interface{}{
			"type":      "document",
			"mimetype":  evtMsg.DocumentMessage.Mimetype,
			"file_name": evtMsg.DocumentMessage.FileName,
		}
		if evtMsg.DocumentMessage.URL != "" && strings.HasPrefix(evtMsg.DocumentMessage.URL, "http") {
			att["url"] = evtMsg.DocumentMessage.URL
		}
		if evtMsg.DocumentMessage.Base64 != "" {
			att["base64"] = evtMsg.DocumentMessage.Base64
			if att["url"] == nil {
				att["url"] = fmt.Sprintf("data:%s;base64,%s", evtMsg.DocumentMessage.Mimetype, evtMsg.DocumentMessage.Base64)
			}
		}
		if evtMsg.DocumentMessage.Base64 == "" && evtMsg.DocumentMessage.URL == "" {
			return nil, nil
		}
		attJSON, _ := json.Marshal(att)
		return nil, attJSON
	}

	return nil, nil
}
