package message

import (
	"context"
	"database/sql"

	"centrachannel/internal/src/conversation"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type messageRepository struct{}

func NewMessageRepository() MessageRepository {
	return &messageRepository{}
}

func scanMessage(row interface {
	Scan(dest ...interface{}) error
}) (*Message, error) {
	var m Message
	var text, webhookID, webhookReplyID sql.NullString
	var attachment sql.NullString
	var tenantID uuid.UUID

	err := row.Scan(&m.ID, &tenantID, &text, &attachment, &m.Status, &m.SenderID, &m.SenderType, &webhookID, &webhookReplyID, &m.ConversationID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, err
	}
	m.TenantID = tenantID
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
	return &m, nil
}

func (r *messageRepository) ListCursor(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error) {
	cursorClause := ""
	args := []interface{}{conversationID, tenantID}
	argIdx := 3

	if lastID != uuid.Nil {
		cursorClause = fmt.Sprintf(" AND (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $%d AND tenant_id = $2 AND conversation_id = $1)", argIdx)
		args = append(args, lastID)
		argIdx++
	}

	args = append(args, limit)

	rows, err := q.QueryContext(ctx, fmt.Sprintf(`SELECT id, tenant_id, text, attachment, status, sender_id, sender_type, webhook_message_id, webhook_message_reply_id, conversation_id, created_at, updated_at FROM messages WHERE conversation_id = $1 AND tenant_id = $2%s ORDER BY created_at DESC, id DESC LIMIT $%d`, cursorClause, argIdx), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

func (r *messageRepository) Create(ctx context.Context, db *sql.DB, msg *Message, lastMessageJSON []byte, lastAgentID *uuid.UUID, conversations conversation.ConversationRepository) (uuid.UUID, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()
	if msg.WebhookMessageID != nil {
		// Serializes duplicate provider events without deleting historical duplicates.
		key := msg.TenantID.String() + ":" + msg.ConversationID.String() + ":" + *msg.WebhookMessageID
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
			return uuid.Nil, err
		}
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM messages WHERE tenant_id=$1 AND conversation_id=$2 AND webhook_message_id=$3)`, msg.TenantID, msg.ConversationID, *msg.WebhookMessageID).Scan(&exists); err != nil {
			return uuid.Nil, err
		}
		if exists {
			return uuid.Nil, nil
		}
	}
	var id uuid.UUID
	err = tx.QueryRowContext(ctx, `INSERT INTO messages (tenant_id, text, attachment, status, sender_id, sender_type, webhook_message_id, webhook_message_reply_id, conversation_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		msg.TenantID, msg.Text, msg.Attachment, msg.Status, msg.SenderID, msg.SenderType,
		msg.WebhookMessageID, msg.WebhookMessageReplyID, msg.ConversationID,
		time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	if conversations != nil {
		if err := conversations.UpdateLastMessage(ctx, tx, msg.TenantID, msg.ConversationID, lastMessageJSON, lastAgentID); err != nil {
			return uuid.Nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *messageRepository) UpdateStatusByWebhookID(ctx context.Context, q DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
	result, err := q.ExecContext(ctx, `UPDATE messages SET status=$1, updated_at=$2 WHERE webhook_message_id=$3 AND tenant_id=$4 AND CASE $1 WHEN 'sent' THEN status IN ('pending','sent') WHEN 'delivered' THEN status IN ('pending','sent','delivered') WHEN 'failed' THEN status IN ('pending','sent','failed') ELSE true END`, status, time.Now(), webhookMessageID, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found")
	}
	return nil
}

func (r *messageRepository) UpdateWebhookID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, webhookMessageID string) error {
	_, err := q.ExecContext(ctx, `UPDATE messages SET webhook_message_id=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4`, webhookMessageID, time.Now(), id, tenantID)
	return err
}

func (r *messageRepository) UpdateStatus(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	result, err := q.ExecContext(ctx, `UPDATE messages SET status=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4 AND CASE $1 WHEN 'sent' THEN status IN ('pending','sent') WHEN 'failed' THEN status IN ('pending','sent','failed') ELSE true END`, status, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 && status != "sent" && status != "failed" {
		return fmt.Errorf("message not found")
	}
	return nil
}

func (r *messageRepository) RetryFailed(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, senderID uuid.UUID) (*Message, error) {
	return scanMessage(q.QueryRowContext(ctx, `UPDATE messages SET status='pending', updated_at=$1 WHERE id=$2 AND tenant_id=$3 AND sender_id=$4 AND sender_type='user' AND status='failed' AND webhook_message_id IS NULL RETURNING id, tenant_id, text, attachment, status, sender_id, sender_type, webhook_message_id, webhook_message_reply_id, conversation_id, created_at, updated_at`, time.Now(), id, tenantID, senderID))
}

func (r *messageRepository) MarkReadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) error {
	_, err := q.ExecContext(ctx,
		`UPDATE messages SET status='read', updated_at=$1 WHERE conversation_id=$2 AND tenant_id=$3 AND sender_type='contact' AND status!='read'`,
		time.Now(), conversationID, tenantID,
	)
	return err
}

func (r *messageRepository) GetUnreadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]*Message, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, tenant_id, text, attachment, status, sender_id, sender_type, webhook_message_id, webhook_message_reply_id, conversation_id, created_at, updated_at FROM messages WHERE conversation_id=$1 AND tenant_id=$2 AND sender_type='contact' AND webhook_message_id IS NOT NULL AND status!='read' ORDER BY created_at ASC, id ASC`,
		conversationID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}
