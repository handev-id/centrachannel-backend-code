package message

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type messageRepository struct{}

func NewMessageRepository() MessageRepository {
	return &messageRepository{}
}

func scanMessage(row interface{ Scan(dest ...interface{}) error }) (*Message, error) {
	var m Message
	var text, webhookID, webhookReplyID sql.NullString
	var attachment sql.NullString
	var tenantID int

	err := row.Scan(&m.ID, &tenantID, &text, &attachment, &m.Status, &m.SenderID, &m.SenderType, &webhookID, &webhookReplyID, &m.ConversationID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, err
	}
	m.TenantID = tenantID
	if text.Valid { m.Text = &text.String }
	if attachment.Valid { m.Attachment = []byte(attachment.String) }
	if webhookID.Valid { m.WebhookMessageID = &webhookID.String }
	if webhookReplyID.Valid { m.WebhookMessageReplyID = &webhookReplyID.String }
	return &m, nil
}

func (r *messageRepository) List(ctx context.Context, q DBTX, conversationID int, limit, offset int) ([]*Message, int, error) {
	countQuery := `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`
	var total int
	if err := q.QueryRowContext(ctx, countQuery, conversationID).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*Message{}, 0, nil
	}

	query := `SELECT id, tenant_id, text, attachment, status, sender_id, sender_type, webhook_message_id, webhook_message_reply_id, conversation_id, created_at, updated_at FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`
	rows, err := q.QueryContext(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, 0, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, total, rows.Err()
}

func (r *messageRepository) Create(ctx context.Context, q DBTX, msg *Message) (int, error) {
	query := `INSERT INTO messages (tenant_id, text, attachment, status, sender_id, sender_type, webhook_message_id, webhook_message_reply_id, conversation_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query,
		msg.TenantID, msg.Text, msg.Attachment, msg.Status, msg.SenderID, msg.SenderType,
		msg.WebhookMessageID, msg.WebhookMessageReplyID, msg.ConversationID,
		time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *messageRepository) UpdateStatus(ctx context.Context, q DBTX, id int, status string) error {
	result, err := q.ExecContext(ctx, `UPDATE messages SET status=$1, updated_at=$2 WHERE id=$3`, status, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found")
	}
	return nil
}
