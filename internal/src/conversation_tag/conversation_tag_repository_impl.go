package conversation_tag

import (
	"context"
	"fmt"
	"time"
)

type conversationTagRepository struct{}

func NewConversationTagRepository() ConversationTagRepository {
	return &conversationTagRepository{}
}

func (r *conversationTagRepository) ListByConversation(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]ConversationTag, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, conversation_id, tag_id, created_at, updated_at FROM conversation_tag WHERE tenant_id = $1 AND conversation_id = $2`, tenantID, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []ConversationTag
	for rows.Next() {
		var ct ConversationTag
		if err := rows.Scan(&ct.ID, &ct.TenantID, &ct.ConversationID, &ct.TagID, &ct.CreatedAt, &ct.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, ct)
	}
	return tags, rows.Err()
}

func (r *conversationTagRepository) Attach(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error {
	_, err := q.ExecContext(ctx, `INSERT INTO conversation_tag (tenant_id, conversation_id, tag_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (conversation_id, tag_id) DO NOTHING`, tenantID, conversationID, tagID, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to attach tag: %w", err)
	}
	return nil
}

func (r *conversationTagRepository) Detach(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error {
	result, err := q.ExecContext(ctx, `DELETE FROM conversation_tag WHERE tenant_id = $1 AND conversation_id = $2 AND tag_id = $3`, tenantID, conversationID, tagID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tag not attached to conversation")
	}
	return nil
}
