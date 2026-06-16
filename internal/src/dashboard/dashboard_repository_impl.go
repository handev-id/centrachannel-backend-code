package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type dashboardRepository struct{}

func NewDashboardRepository() DashboardRepository {
	return &dashboardRepository{}
}

func (r *dashboardRepository) GetTotalContacts(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM contacts WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetNewContacts(ctx context.Context, q DBTX, tenantID int, days int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM contacts WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= NOW() - ($2 || ' days')::INTERVAL`, tenantID, days).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetTotalConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetActiveConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND status != 'resolved'`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetNewConversations(ctx context.Context, q DBTX, tenantID int, days int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND created_at >= NOW() - ($2 || ' days')::INTERVAL`, tenantID, days).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetResolvedToday(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND status = 'resolved' AND updated_at >= $2`, tenantID, time.Now().Truncate(24*time.Hour)).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetUnassignedCount(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND status = 'unassigned'`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetTotalUnreadConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND unread_count > 0`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetTotalUnreadMessages(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var total sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT COALESCE(SUM(unread_count), 0) FROM conversations WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return int(total.Int64), nil
}

func (r *dashboardRepository) GetTotalMessages(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetMessagesToday(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE tenant_id = $1 AND created_at >= $2`, tenantID, time.Now().Truncate(24*time.Hour)).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetTotalCampaigns(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM campaigns WHERE tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetConnectedWhatsappDevices(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM whatsapp_devices WHERE tenant_id = $1 AND status = 'CONNECTED'`, tenantID).Scan(&count)
	return count, err
}

func (r *dashboardRepository) GetConversationsByStatus(ctx context.Context, q DBTX, tenantID int) (*ConversationsByStatus, error) {
	rows, err := q.QueryContext(ctx, `SELECT status, COUNT(*) as total FROM conversations WHERE tenant_id = $1 GROUP BY status`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &ConversationsByStatus{}
	for rows.Next() {
		var status string
		var total int
		if err := rows.Scan(&status, &total); err != nil {
			return nil, err
		}
		switch status {
		case "unassigned":
			result.Unassigned = total
		case "assigned":
			result.Assigned = total
		case "resolved":
			result.Resolved = total
		}
	}
	return result, rows.Err()
}

func (r *dashboardRepository) GetConversationsByChannel(ctx context.Context, q DBTX, tenantID int) ([]ConversationsByChannel, error) {
	query := `
		SELECT c.channel_id, ch.name, COUNT(*) as total, COALESCE(SUM(c.unread_count), 0) as unread
		FROM conversations c
		JOIN channels ch ON ch.id = c.channel_id
		WHERE c.tenant_id = $1
		GROUP BY c.channel_id, ch.name
		ORDER BY total DESC
	`
	rows, err := q.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ConversationsByChannel
	for rows.Next() {
		var item ConversationsByChannel
		if err := rows.Scan(&item.ChannelID, &item.ChannelName, &item.Total, &item.Unread); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

type recentConversationScan struct {
	ID               int
	Status           string
	UnreadCount      int
	LastActivity     sql.NullTime
	LastMessage      sql.NullString
	ChannelID        int
	ChannelName      string
	ChannelLogo      sql.NullString
	AgentID          sql.NullInt64
	AgentFirstName   sql.NullString
	AgentLastName    sql.NullString
	AgentAvatar      sql.NullString
	ContactID        int
	ContactFirstName string
	ContactLastName  sql.NullString
	ContactAvatar    sql.NullString
}

func (r *dashboardRepository) GetRecentConversations(ctx context.Context, q DBTX, tenantID int, limit int) ([]RecentConversation, error) {
	query := `
		SELECT
			c.id, c.status, c.unread_count, c.last_activity, c.last_message,
			ch.id, ch.name, ch.logo,
			u.id, u.first_name, u.last_name, u.avatar,
			ct.id, ct.first_name, ct.last_name, ct.avatar
		FROM conversations c
		JOIN channels ch ON ch.id = c.channel_id
		JOIN profiles p ON p.id = c.profile_id AND p.deleted_at IS NULL
		JOIN contacts ct ON ct.id = p.contact_id AND ct.deleted_at IS NULL
		LEFT JOIN users u ON u.id = c.agent_id AND u.deleted_at IS NULL
		WHERE c.tenant_id = $1
		ORDER BY c.last_activity DESC NULLS LAST, c.id DESC
		LIMIT $2
	`
	rows, err := q.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RecentConversation
	for rows.Next() {
		var s recentConversationScan
		if err := rows.Scan(
			&s.ID, &s.Status, &s.UnreadCount, &s.LastActivity, &s.LastMessage,
			&s.ChannelID, &s.ChannelName, &s.ChannelLogo,
			&s.AgentID, &s.AgentFirstName, &s.AgentLastName, &s.AgentAvatar,
			&s.ContactID, &s.ContactFirstName, &s.ContactLastName, &s.ContactAvatar,
		); err != nil {
			return nil, err
		}

		item := RecentConversation{
			ID:          s.ID,
			Status:      s.Status,
			UnreadCount: s.UnreadCount,
			Channel: &ChannelBrief{
				ID:   s.ChannelID,
				Name: s.ChannelName,
			},
			Contact: &ContactBrief{
				ID:        s.ContactID,
				FirstName: s.ContactFirstName,
			},
		}

		if s.LastActivity.Valid {
			t := s.LastActivity.Time.Format(time.RFC3339)
			item.LastActivity = &t
		}
		if s.LastMessage.Valid {
			item.LastMessage = json.RawMessage(s.LastMessage.String)
		}
		if s.ChannelLogo.Valid {
			item.Channel.Logo = json.RawMessage(s.ChannelLogo.String)
		}
		if s.ContactLastName.Valid {
			item.Contact.LastName = &s.ContactLastName.String
		}
		if s.ContactAvatar.Valid {
			item.Contact.Avatar = json.RawMessage(s.ContactAvatar.String)
		}
		if s.AgentID.Valid {
			item.Agent = &AgentBrief{
				ID:        int(s.AgentID.Int64),
				FirstName: s.AgentFirstName.String,
			}
			if s.AgentLastName.Valid {
				item.Agent.LastName = &s.AgentLastName.String
			}
			if s.AgentAvatar.Valid {
				item.Agent.Avatar = json.RawMessage(s.AgentAvatar.String)
			}
		}

		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *dashboardRepository) GetConversationChart(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error) {
	query := `SELECT DATE(created_at) as date, COUNT(*) as count FROM conversations WHERE tenant_id = $1 AND created_at >= NOW() - ($2 || ' days')::INTERVAL GROUP BY DATE(created_at) ORDER BY date`
	rows, err := q.QueryContext(ctx, query, tenantID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []ChartDataPoint
	for rows.Next() {
		var dp ChartDataPoint
		var date time.Time
		if err := rows.Scan(&date, &dp.Count); err != nil {
			return nil, err
		}
		dp.Date = date.Format("2006-01-02")
		data = append(data, dp)
	}
	return data, rows.Err()
}

func (r *dashboardRepository) GetContactsByStatus(ctx context.Context, q DBTX, tenantID int) (*ContactsByStatus, error) {
	rows, err := q.QueryContext(ctx, `SELECT status, COUNT(*) as total FROM contacts WHERE tenant_id = $1 AND deleted_at IS NULL GROUP BY status`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &ContactsByStatus{}
	for rows.Next() {
		var status string
		var total int
		if err := rows.Scan(&status, &total); err != nil {
			return nil, err
		}
		switch status {
		case "individual":
			result.Individual = total
		case "institution":
			result.Institution = total
		}
	}
	return result, rows.Err()
}

func (r *dashboardRepository) GetMessagesBySenderType(ctx context.Context, q DBTX, tenantID int) (*MessagesBySenderType, error) {
	rows, err := q.QueryContext(ctx, `SELECT sender_type, COUNT(*) as total FROM messages WHERE tenant_id = $1 GROUP BY sender_type`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &MessagesBySenderType{}
	for rows.Next() {
		var senderType string
		var total int
		if err := rows.Scan(&senderType, &total); err != nil {
			return nil, err
		}
		switch senderType {
		case "contact":
			result.Contact = total
		case "user":
			result.User = total
		}
	}
	return result, rows.Err()
}

func (r *dashboardRepository) GetMessagesByDate(ctx context.Context, q DBTX, tenantID int, days int) ([]MessagesByDate, error) {
	query := `SELECT DATE(created_at) as date, COUNT(*) as count FROM messages WHERE tenant_id = $1 AND created_at >= NOW() - ($2 || ' days')::INTERVAL GROUP BY DATE(created_at) ORDER BY date`
	rows, err := q.QueryContext(ctx, query, tenantID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []MessagesByDate
	for rows.Next() {
		var item MessagesByDate
		var date time.Time
		if err := rows.Scan(&date, &item.Count); err != nil {
			return nil, err
		}
		item.Date = date.Format("2006-01-02")
		data = append(data, item)
	}
	return data, rows.Err()
}

func (r *dashboardRepository) GetCampaignsByStatus(ctx context.Context, q DBTX, tenantID int) (map[string]int, error) {
	rows, err := q.QueryContext(ctx, `SELECT status, COUNT(*) as total FROM campaigns WHERE tenant_id = $1 GROUP BY status`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var total int
		if err := rows.Scan(&status, &total); err != nil {
			return nil, err
		}
		result[status] = total
	}
	return result, rows.Err()
}

type agentWorkloadScan struct {
	ID        int
	FirstName string
	LastName  sql.NullString
	Avatar    sql.NullString
	Ongoing   int
}

func (r *dashboardRepository) GetAgentWorkload(ctx context.Context, q DBTX, tenantID int) ([]AgentWorkload, error) {
	query := `
		SELECT u.id, u.first_name, u.last_name, u.avatar, COUNT(c.id) as ongoing
		FROM users u
		INNER JOIN conversations c ON c.agent_id = u.id AND c.tenant_id = u.tenant_id AND c.status != 'resolved'
		WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
		GROUP BY u.id, u.first_name, u.last_name, u.avatar
		ORDER BY ongoing DESC
	`
	rows, err := q.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentWorkload
	for rows.Next() {
		var s agentWorkloadScan
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Avatar, &s.Ongoing); err != nil {
			return nil, err
		}
		item := AgentWorkload{
			ID:                   s.ID,
			FirstName:            s.FirstName,
			OngoingConversations: s.Ongoing,
		}
		if s.LastName.Valid {
			item.LastName = &s.LastName.String
		}
		if s.Avatar.Valid {
			item.Avatar = json.RawMessage(s.Avatar.String)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
