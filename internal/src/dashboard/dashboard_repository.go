package dashboard

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type DashboardRepository interface {
	GetTotalContacts(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetNewContacts(ctx context.Context, q DBTX, tenantID int, days int) (int, error)
	GetTotalConversations(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetActiveConversations(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetNewConversations(ctx context.Context, q DBTX, tenantID int, days int) (int, error)
	GetResolvedToday(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetUnassignedCount(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetTotalUnreadConversations(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetTotalUnreadMessages(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetTotalMessages(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetMessagesToday(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetTotalCampaigns(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetConnectedWhatsappDevices(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetConversationsByStatus(ctx context.Context, q DBTX, tenantID int) (*ConversationsByStatus, error)
	GetConversationsByChannel(ctx context.Context, q DBTX, tenantID int) ([]ConversationsByChannel, error)
	GetRecentConversations(ctx context.Context, q DBTX, tenantID int, limit int) ([]RecentConversation, error)
	GetConversationChart(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error)
	GetContactsByStatus(ctx context.Context, q DBTX, tenantID int) (*ContactsByStatus, error)
	GetMessagesBySenderType(ctx context.Context, q DBTX, tenantID int) (*MessagesBySenderType, error)
	GetMessagesByDate(ctx context.Context, q DBTX, tenantID int, days int) ([]MessagesByDate, error)
	GetCampaignsByStatus(ctx context.Context, q DBTX, tenantID int) (map[string]int, error)
	GetAgentWorkload(ctx context.Context, q DBTX, tenantID int) ([]AgentWorkload, error)
}
