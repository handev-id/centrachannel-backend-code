package message

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"strings"

	"github.com/google/uuid"

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

func validateReplyTarget(quote *Quote) error {
	if len(quote.Attachment) > 0 && string(quote.Attachment) != "null" {
		var att Attachment
		if err := json.Unmarshal(quote.Attachment, &att); err != nil {
			return err
		}
		switch att.Type {
		case "image", "video", "audio", "document", "sticker":
			return nil
		}
		return fmt.Errorf("cannot quote unsupported message type")
	}
	if quote.Text == nil {
		return fmt.Errorf("reply target has no supported content")
	}
	return nil
}

var ErrMessageNotFound = errors.New("message not found")

func scanMessage(row interface {
	Scan(dest ...interface{}) error
}) (*Message, error) {
	var m Message
	var text, webhookID, webhookReplyID sql.NullString
	var attachment sql.NullString
	var quote, reactions sql.NullString
	var editedAt sql.NullTime
	var tenantID uuid.UUID

	err := row.Scan(&m.ID, &tenantID, &text, &attachment, &m.Status, &m.SenderID, &m.SenderType, &webhookID, &webhookReplyID, &m.ConversationID, &m.CreatedAt, &m.UpdatedAt, &editedAt, &quote, &reactions)
	if err == sql.ErrNoRows {
		return nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, err
	}
	m.TenantID = tenantID
	m.Reactions = []Reaction{}
	if reactions.Valid {
		if err := json.Unmarshal([]byte(reactions.String), &m.Reactions); err != nil {
			return nil, err
		}
	}
	if text.Valid {
		m.Text = &text.String
	}
	if attachment.Valid {
		m.Attachment = []byte(attachment.String)
	}
	if webhookID.Valid {
		m.WebhookMessageID = &webhookID.String
	}
	if webhookReplyID.Valid {
		m.WebhookMessageReplyID = &webhookReplyID.String
	}
	if editedAt.Valid {
		m.EditedAt = &editedAt.Time
		m.IsEdited = true
	}
	if quote.Valid {
		if err := json.Unmarshal([]byte(quote.String), &m.Reply); err != nil {
			return nil, err
		}
	}
	return &m, nil
}
