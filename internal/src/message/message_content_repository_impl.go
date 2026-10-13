package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func (r *messageRepository) GetByID(ctx context.Context, q DBTX, tenantID, conversationID, id uuid.UUID) (*Message, error) {
	return scanMessage(q.QueryRowContext(ctx, `SELECT `+messageSelectColumns+` FROM messages WHERE tenant_id=$1 AND conversation_id=$2 AND id=$3`, tenantID, conversationID, id))
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
		_, err = tx.ExecContext(ctx, `UPDATE conversations SET last_message=jsonb_set(jsonb_set(COALESCE(last_message,'{}'::jsonb),'{text}',COALESCE(to_jsonb($1::text),'null'::jsonb)),'{attachment}',COALESCE($2::jsonb,'null'::jsonb)),updated_at=clock_timestamp()
          WHERE tenant_id=$3 AND id=$4 AND last_message->>'id'=$5`, target.Text, []byte(target.Attachment), tenantID, target.ConversationID, target.ID.String())
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return target, nil
}

func (r *messageRepository) GetByProviderID(ctx context.Context, q DBTX, tenantID, profileID, channelID uuid.UUID, providerID string) (*Message, error) {
	return scanMessage(q.QueryRowContext(ctx, `SELECT `+messageSelectColumns+` FROM messages JOIN conversations c ON c.id=messages.conversation_id AND c.tenant_id=messages.tenant_id
 WHERE messages.tenant_id=$1 AND c.profile_id=$2 AND c.channel_id=$3 AND messages.webhook_message_id=$4 ORDER BY messages.created_at,messages.id LIMIT 1`, tenantID, profileID, channelID, providerID))
}

func (r *messageRepository) UpdateWebhookID(ctx context.Context, db *sql.DB, tenantID, id uuid.UUID, providerID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var profileID, channelID, conversationID uuid.UUID
	if err = tx.QueryRowContext(ctx, `SELECT c.profile_id,c.channel_id,c.id FROM messages m JOIN conversations c ON c.id=m.conversation_id AND c.tenant_id=m.tenant_id WHERE m.id=$1 AND m.tenant_id=$2`, id, tenantID).Scan(&profileID, &channelID, &conversationID); err != nil {
		return err
	}
	key := tenantID.String() + ":" + profileID.String() + ":" + channelID.String() + ":" + providerID
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE messages SET webhook_message_id=$1,updated_at=clock_timestamp() WHERE id=$2 AND tenant_id=$3`, providerID, id, tenantID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE messages m SET text=e.payload->>'text',
 attachment=CASE WHEN m.attachment->>'type' IN ('image','video','document') THEN jsonb_set(m.attachment,'{caption}',to_jsonb(COALESCE(e.payload->>'text',''))) ELSE m.attachment END,
 edited_at=e.event_at,updated_at=clock_timestamp() FROM message_mutations e
 WHERE m.id=$1 AND m.tenant_id=$2 AND e.tenant_id=$2 AND e.profile_id=$3 AND e.channel_id=$4 AND e.target_provider_id=m.webhook_message_id AND e.kind='edit' AND e.payload->>'sender_type'=m.sender_type AND (m.edited_at IS NULL OR m.edited_at<=e.event_at)`, id, tenantID, profileID, channelID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE conversations c SET last_message=jsonb_set(jsonb_set(c.last_message,'{text}',COALESCE(to_jsonb(m.text),'null'::jsonb)),'{attachment}',COALESCE(m.attachment,'null'::jsonb)),updated_at=clock_timestamp() FROM messages m
 WHERE c.id=$1 AND c.tenant_id=$2 AND m.id=$3 AND m.tenant_id=$2 AND m.edited_at IS NOT NULL AND c.last_message->>'id'=m.id::text`, conversationID, tenantID, id); err != nil {
		return err
	}
	return tx.Commit()
}
