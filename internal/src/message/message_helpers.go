package message

import (
	"encoding/json"
	"errors"
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

func buildProviderQuote(quote *Quote, remoteJID string) (json.RawMessage, error) {
	content := map[string]interface{}{}
	if len(quote.Attachment) > 0 && string(quote.Attachment) != "null" {
		var att Attachment
		if err := json.Unmarshal(quote.Attachment, &att); err != nil {
			return nil, err
		}
		names := map[string]string{"image": "imageMessage", "video": "videoMessage", "audio": "audioMessage", "document": "documentMessage", "sticker": "stickerMessage"}
		name, ok := names[att.Type]
		if !ok {
			return nil, fmt.Errorf("cannot quote unsupported message type")
		}
		content[name] = map[string]interface{}{"mimetype": att.Mimetype, "caption": att.Caption, "fileName": att.FileName, "isAnimated": att.IsAnimated}
	} else if quote.Text != nil {
		content["conversation"] = *quote.Text
	} else {
		return nil, fmt.Errorf("reply target has no supported content")
	}
	return json.Marshal(map[string]interface{}{"key": map[string]interface{}{"id": quote.ProviderMessageID, "remoteJid": remoteJID, "fromMe": quote.SenderType != "contact"}, "message": content})
}

// The indexed mutation projection keeps message/quote/reaction reads in one query.
const messageSelectColumns = `messages.id,messages.tenant_id,messages.text,messages.attachment,messages.status,messages.sender_id,messages.sender_type,messages.webhook_message_id,messages.webhook_message_reply_id,messages.conversation_id,messages.created_at,messages.updated_at,messages.edited_at,messages.quote,
 COALESCE((SELECT jsonb_agg(jsonb_build_object('actor_key',e.actor_key,'sender_type',e.payload->>'sender_type','emoji',e.payload->>'emoji','reacted_at',e.event_at) ORDER BY e.actor_key)
 FROM message_mutations e JOIN conversations c ON c.profile_id=e.profile_id AND c.channel_id=e.channel_id AND c.tenant_id=e.tenant_id
 WHERE c.id=messages.conversation_id AND e.tenant_id=messages.tenant_id AND e.target_provider_id=messages.webhook_message_id AND e.kind='reaction' AND COALESCE(e.payload->>'emoji','')<>''),'[]'::jsonb)`

var ErrMessageNotFound = errors.New("message not found")
