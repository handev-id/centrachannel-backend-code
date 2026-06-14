package conversation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type conversationRepository struct{}

func NewConversationRepository() ConversationRepository {
	return &conversationRepository{}
}

func scanConversation(row interface{ Scan(dest ...interface{}) error }) (*Conversation, error) {
	var c Conversation
	var agentID, lastAgentID sql.NullInt64
	var lastMessage sql.NullString
	var lastActivity, lastSeen sql.NullTime
	var tenantID int

	err := row.Scan(&c.ID, &tenantID, &c.Status, &c.ProfileID, &agentID, &c.ChannelID, &lastAgentID, &c.UnreadCount, &lastMessage, &lastActivity, &lastSeen, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, err
	}
	c.TenantID = tenantID
	if agentID.Valid { id := int(agentID.Int64); c.AgentID = &id }
	if lastAgentID.Valid { id := int(lastAgentID.Int64); c.LastAgentID = &id }
	if lastMessage.Valid { c.LastMessage = []byte(lastMessage.String) }
	if lastActivity.Valid { c.LastActivity = &lastActivity.Time }
	if lastSeen.Valid { c.LastSeen = &lastSeen.Time }
	return &c, nil
}

func (r *conversationRepository) List(ctx context.Context, q DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*Conversation, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("c.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if channelID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.channel_id = $%d", argIdx))
		args = append(args, channelID)
		argIdx++
	}

	if agentID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.agent_id = $%d", argIdx))
		args = append(args, agentID)
		argIdx++
	}

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(p.display_name) LIKE LOWER($%d) OR LOWER(p.username) LIKE LOWER($%d))", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	fromClause := "conversations c"
	if search != "" {
		fromClause = "conversations c INNER JOIN profiles p ON p.id = c.profile_id"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", fromClause, where)
	var total int
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*Conversation{}, 0, nil
	}

	dataQuery := fmt.Sprintf(`SELECT c.id, c.tenant_id, c.status, c.profile_id, c.agent_id, c.channel_id, c.last_agent_id, c.unread_count, c.last_message, c.last_activity, c.last_seen, c.created_at, c.updated_at FROM %s WHERE %s ORDER BY c.last_activity DESC NULLS LAST LIMIT $%d OFFSET $%d`, fromClause, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := q.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var convs []*Conversation
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, 0, err
		}
		convs = append(convs, conv)
	}
	return convs, total, rows.Err()
}

func (r *conversationRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Conversation, error) {
	query := `SELECT id, tenant_id, status, profile_id, agent_id, channel_id, last_agent_id, unread_count, last_message, last_activity, last_seen, created_at, updated_at FROM conversations WHERE id = $1 AND tenant_id = $2`
	return scanConversation(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *conversationRepository) Create(ctx context.Context, q DBTX, conv *Conversation) (int, error) {
	query := `INSERT INTO conversations (tenant_id, status, profile_id, agent_id, channel_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query, conv.TenantID, conv.Status, conv.ProfileID, conv.AgentID, conv.ChannelID, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *conversationRepository) UpdateStatus(ctx context.Context, q DBTX, tenantID int, id int, status string) error {
	result, err := q.ExecContext(ctx, `UPDATE conversations SET status=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4`, status, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

func (r *conversationRepository) Assign(ctx context.Context, q DBTX, tenantID int, id int, agentID int) error {
	result, err := q.ExecContext(ctx, `UPDATE conversations SET agent_id=$1, status='assigned', updated_at=$2 WHERE id=$3 AND tenant_id=$4`, agentID, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

func (r *conversationRepository) Unassign(ctx context.Context, q DBTX, tenantID int, id int) error {
	result, err := q.ExecContext(ctx, `UPDATE conversations SET agent_id=NULL, status='unassigned', updated_at=$1 WHERE id=$2 AND tenant_id=$3`, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

func (r *conversationRepository) MarkRead(ctx context.Context, q DBTX, tenantID int, id int) error {
	result, err := q.ExecContext(ctx, `UPDATE conversations SET unread_count=0, last_seen=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4`, time.Now(), time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

func (r *conversationRepository) UpdateLastMessage(ctx context.Context, q DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID int) error {
	result, err := q.ExecContext(ctx, `UPDATE conversations SET last_message=$1, last_activity=$2, unread_count=unread_count+1, last_agent_id=$3, updated_at=$4 WHERE id=$5 AND tenant_id=$6`, lastMessageJSON, time.Now(), lastAgentID, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}
