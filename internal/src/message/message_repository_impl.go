package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"centrachannel/internal/src/conversation"
)

// The indexed mutation projection keeps message/quote/reaction reads in one query.
const messageSelectColumns = `messages.id,messages.tenant_id,messages.text,messages.attachment,messages.status,messages.sender_id,messages.sender_type,messages.webhook_message_id,messages.webhook_message_reply_id,messages.conversation_id,messages.created_at,messages.updated_at,messages.edited_at,messages.quote,
 COALESCE((SELECT jsonb_agg(jsonb_build_object('actor_key',e.actor_key,'sender_type',e.payload->>'sender_type','emoji',e.payload->>'emoji','reacted_at',e.event_at) ORDER BY e.actor_key)
 FROM message_mutations e JOIN conversations c ON c.profile_id=e.profile_id AND c.channel_id=e.channel_id AND c.tenant_id=e.tenant_id
 WHERE c.id=messages.conversation_id AND e.tenant_id=messages.tenant_id AND e.target_provider_id=messages.webhook_message_id AND e.kind='reaction' AND COALESCE(e.payload->>'emoji','')<>''),'[]'::jsonb)`

type messageRepository struct{}

func NewMessageRepository() MessageRepository {
	return &messageRepository{}
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

	rows, err := q.QueryContext(ctx, fmt.Sprintf(`SELECT `+messageSelectColumns+` FROM messages WHERE conversation_id = $1 AND tenant_id = $2%s ORDER BY created_at DESC, id DESC LIMIT $%d`, cursorClause, argIdx), args...)
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
	var profileID, channelID uuid.UUID
	if err = tx.QueryRowContext(ctx, `SELECT profile_id,channel_id FROM conversations WHERE tenant_id=$1 AND id=$2`, msg.TenantID, msg.ConversationID).Scan(&profileID, &channelID); err != nil {
		return uuid.Nil, err
	}
	if msg.WebhookMessageID != nil {
		// Serializes duplicate provider events without deleting historical duplicates.
		key := msg.TenantID.String() + ":" + profileID.String() + ":" + channelID.String() + ":" + *msg.WebhookMessageID
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
	quote, err := json.Marshal(msg.Reply)
	if err != nil {
		return uuid.Nil, err
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO messages (tenant_id, text, attachment, status, sender_id, sender_type, webhook_message_id, webhook_message_reply_id, conversation_id, created_at, updated_at, quote) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		msg.TenantID, msg.Text, msg.Attachment, msg.Status, msg.SenderID, msg.SenderType,
		msg.WebhookMessageID, msg.WebhookMessageReplyID, msg.ConversationID,
		time.Now(), time.Now(), quote,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	if err = r.applyPendingEdit(ctx, tx, msg.TenantID, profileID, channelID, id); err != nil {
		return uuid.Nil, err
	}
	stored, err := scanMessage(tx.QueryRowContext(ctx, `SELECT `+messageSelectColumns+` FROM messages WHERE id=$1 AND tenant_id=$2`, id, msg.TenantID))
	if err != nil {
		return uuid.Nil, err
	}
	*msg = *stored
	if conversations != nil {
		lastMessageJSON, err = json.Marshal(map[string]interface{}{"id": id, "text": msg.Text, "attachment": msg.Attachment, "sender_type": msg.SenderType})
		if err != nil {
			return uuid.Nil, err
		}
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
	return scanMessage(q.QueryRowContext(ctx, `UPDATE messages SET status='pending', updated_at=$1 WHERE id=$2 AND tenant_id=$3 AND sender_id=$4 AND sender_type='user' AND status='failed' AND webhook_message_id IS NULL RETURNING `+messageSelectColumns, time.Now(), id, tenantID, senderID))
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
		`SELECT `+messageSelectColumns+` FROM messages WHERE conversation_id=$1 AND tenant_id=$2 AND sender_type='contact' AND webhook_message_id IS NOT NULL AND status!='read' ORDER BY created_at ASC, id ASC`,
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

func (r *messageRepository) GetByID(ctx context.Context, q DBTX, tenantID, conversationID, id uuid.UUID) (*Message, error) {
	return scanMessage(q.QueryRowContext(ctx, `SELECT `+messageSelectColumns+` FROM messages WHERE tenant_id=$1 AND conversation_id=$2 AND id=$3`, tenantID, conversationID, id))
}

func (r *messageRepository) GetByProviderID(ctx context.Context, q DBTX, tenantID, profileID, channelID uuid.UUID, providerID string) (*Message, error) {
	return scanMessage(q.QueryRowContext(ctx, `SELECT `+messageSelectColumns+` FROM messages JOIN conversations c ON c.id=messages.conversation_id AND c.tenant_id=messages.tenant_id
 WHERE messages.tenant_id=$1 AND c.profile_id=$2 AND c.channel_id=$3 AND messages.webhook_message_id=$4 ORDER BY messages.created_at,messages.id LIMIT 1`, tenantID, profileID, channelID, providerID))
}

func (r *messageRepository) ApplyMutation(ctx context.Context, db *sql.DB, tenantID, profileID, channelID uuid.UUID, mutation Mutation) (*Message, error) {
	if (mutation.Kind != "edit" && mutation.Kind != "reaction") || mutation.TargetProviderID == "" || mutation.EventAt.IsZero() || mutation.ActorKey == "" {
		return nil, fmt.Errorf("invalid message mutation")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	key := tenantID.String() + ":" + profileID.String() + ":" + channelID.String() + ":" + mutation.TargetProviderID
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(mutation)
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO message_mutations(tenant_id,profile_id,channel_id,target_provider_id,kind,actor_key,payload,event_at,event_id)
      SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE EXISTS(SELECT 1 FROM profiles WHERE id=$2 AND tenant_id=$1)
      ON CONFLICT(tenant_id,profile_id,channel_id,target_provider_id,kind,actor_key) DO UPDATE
      SET payload=EXCLUDED.payload,event_at=EXCLUDED.event_at,event_id=EXCLUDED.event_id
      WHERE (EXCLUDED.event_at,EXCLUDED.event_id) > (message_mutations.event_at,message_mutations.event_id)`, tenantID, profileID, channelID, mutation.TargetProviderID, mutation.Kind, mutation.ActorKey, raw, mutation.EventAt, mutation.EventID)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, tx.Commit()
	}
	if mutation.Kind == "edit" {
		_, err = tx.ExecContext(ctx, `UPDATE messages m SET text=$5,
          attachment=CASE WHEN m.attachment->>'type' IN ('image','video','document') THEN jsonb_set(m.attachment,'{caption}',to_jsonb(COALESCE($5::text,''))) ELSE m.attachment END,
          edited_at=$6,updated_at=clock_timestamp()
          FROM conversations c WHERE m.tenant_id=$1 AND c.tenant_id=$1 AND c.id=m.conversation_id AND c.profile_id=$2 AND c.channel_id=$3
          AND m.webhook_message_id=$4 AND m.sender_type=$7 AND (m.edited_at IS NULL OR m.edited_at<=$6)`, tenantID, profileID, channelID, mutation.TargetProviderID, mutation.Text, mutation.EventAt, mutation.SenderType)
		if err != nil {
			return nil, err
		}
	}
	if mutation.Kind == "reaction" {
		if _, err = tx.ExecContext(ctx, `UPDATE messages SET updated_at=clock_timestamp() WHERE tenant_id=$1 AND webhook_message_id=$4 AND conversation_id IN (SELECT id FROM conversations WHERE tenant_id=$1 AND profile_id=$2 AND channel_id=$3)`, tenantID, profileID, channelID, mutation.TargetProviderID); err != nil {
			return nil, err
		}
	}
	target, err := scanMessage(tx.QueryRowContext(ctx, `SELECT `+messageSelectColumns+` FROM messages JOIN conversations c ON c.id=messages.conversation_id AND c.tenant_id=messages.tenant_id
 WHERE messages.tenant_id=$1 AND c.profile_id=$2 AND c.channel_id=$3 AND messages.webhook_message_id=$4 ORDER BY messages.created_at,messages.id LIMIT 1`, tenantID, profileID, channelID, mutation.TargetProviderID))
	if err != nil {
		// The mutation remains stored and will be applied when its target arrives.
		if !errors.Is(err, ErrMessageNotFound) {
			return nil, err
		}
		return nil, tx.Commit()
	}
	if mutation.Kind == "edit" && target.SenderType == mutation.SenderType {
		if err = r.refreshConversationPreview(ctx, tx, tenantID, target.ID); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return target, nil
}

func (r *messageRepository) UpdateWebhookID(ctx context.Context, db *sql.DB, tenantID, id uuid.UUID, providerID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var profileID, channelID uuid.UUID
	if err = tx.QueryRowContext(ctx, `SELECT c.profile_id,c.channel_id FROM messages m JOIN conversations c ON c.id=m.conversation_id AND c.tenant_id=m.tenant_id WHERE m.id=$1 AND m.tenant_id=$2`, id, tenantID).Scan(&profileID, &channelID); err != nil {
		return err
	}
	key := tenantID.String() + ":" + profileID.String() + ":" + channelID.String() + ":" + providerID
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE messages SET webhook_message_id=$1,updated_at=clock_timestamp() WHERE id=$2 AND tenant_id=$3`, providerID, id, tenantID); err != nil {
		return err
	}
	if err = r.applyPendingEdit(ctx, tx, tenantID, profileID, channelID, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *messageRepository) applyPendingEdit(ctx context.Context, q DBTX, tenantID, profileID, channelID, id uuid.UUID) error {
	if _, err := q.ExecContext(ctx, `UPDATE messages m SET text=e.payload->>'text',
 attachment=CASE WHEN m.attachment->>'type' IN ('image','video','document') THEN jsonb_set(m.attachment,'{caption}',to_jsonb(COALESCE(e.payload->>'text',''))) ELSE m.attachment END,
 edited_at=e.event_at,updated_at=clock_timestamp() FROM message_mutations e
 WHERE m.id=$1 AND m.tenant_id=$2 AND e.tenant_id=$2 AND e.profile_id=$3 AND e.channel_id=$4 AND e.target_provider_id=m.webhook_message_id AND e.kind='edit' AND e.payload->>'sender_type'=m.sender_type AND (m.edited_at IS NULL OR m.edited_at<=e.event_at)`, id, tenantID, profileID, channelID); err != nil {
		return err
	}
	return r.refreshConversationPreview(ctx, q, tenantID, id)
}

func (r *messageRepository) refreshConversationPreview(ctx context.Context, q DBTX, tenantID, id uuid.UUID) error {
	_, err := q.ExecContext(ctx, `UPDATE conversations c SET last_message=jsonb_set(jsonb_set(c.last_message,'{text}',COALESCE(to_jsonb(m.text),'null'::jsonb)),'{attachment}',COALESCE(m.attachment,'null'::jsonb)),updated_at=clock_timestamp() FROM messages m
 WHERE c.id=m.conversation_id AND c.tenant_id=$1 AND m.id=$2 AND m.tenant_id=$1 AND m.edited_at IS NOT NULL AND c.last_message->>'id'=m.id::text`, tenantID, id)
	return err
}
