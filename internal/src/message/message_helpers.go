package message

import (
	"encoding/json"
	"fmt"
	"mime"
	"strings"

	"centrachannel/internal/utils/media"
)

func normalizeMessageContent(req SendMessageRequest) (*string, json.RawMessage, error) {
	var text *string
	if req.Text != nil {
		trimmed := strings.TrimSpace(*req.Text)
		if trimmed != "" {
			text = &trimmed
		}
	}
	if len(req.Attachment) == 0 || string(req.Attachment) == "null" {
		if text == nil {
			return nil, nil, fmt.Errorf("text or attachment is required")
		}
		return text, nil, nil
	}
	var att Attachment
	if err := json.Unmarshal(req.Attachment, &att); err != nil {
		return nil, nil, fmt.Errorf("invalid attachment")
	}
	if !media.IsHTTPURL(att.URL) {
		return nil, nil, fmt.Errorf("attachment URL must be HTTP or HTTPS")
	}
	if att.Type != "image" && att.Type != "video" && att.Type != "audio" && att.Type != "document" {
		return nil, nil, fmt.Errorf("attachment type must be image, video, audio, or document")
	}
	if att.Size <= 0 {
		return nil, nil, fmt.Errorf("attachment size is required")
	}
	if att.Size > media.MaxFileSize {
		return nil, nil, media.ErrTooLarge
	}
	mimetype, _, err := mime.ParseMediaType(att.Mimetype)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment mimetype is required")
	}
	if att.Type != "document" && media.Type(mimetype) != att.Type {
		return nil, nil, fmt.Errorf("attachment type does not match its mimetype")
	}
	att.Mimetype = mimetype
	if text != nil {
		att.Caption = *text
	} else if strings.TrimSpace(att.Caption) != "" {
		caption := strings.TrimSpace(att.Caption)
		text = &caption
		att.Caption = caption
	}
	att.Status = "ready"
	if att.Type == "audio" && text != nil {
		return nil, nil, fmt.Errorf("audio does not support captions; send text separately")
	}
	raw, err := json.Marshal(att)
	return text, raw, err
}
