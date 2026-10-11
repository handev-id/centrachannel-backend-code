package dashboard

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type DashboardRepository interface {
	GetTotalContacts(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetNewContacts(ctx context.Context, q DBTX, tenantID uuid.UUID, days int) (int, error)
	GetTotalConversations(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetActiveConversations(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetNewConversations(ctx context.Context, q DBTX, tenantID uuid.UUID, days int) (int, error)
	GetResolvedToday(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetUnassignedCount(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetTotalUnreadConversations(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetTotalUnreadMessages(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetTotalMessages(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetMessagesToday(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetTotalCampaigns(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetConnectedWhatsappDevices(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
	GetConversationsByStatus(ctx context.Context, q DBTX, tenantID uuid.UUID) (*ConversationsByStatus, error)
	GetConversationsByChannel(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]ConversationsByChannel, error)
	GetRecentConversations(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int) ([]RecentConversation, error)
	GetConversationChart(ctx context.Context, q DBTX, tenantID uuid.UUID, days int) ([]ChartDataPoint, error)
	GetContactsByStatus(ctx context.Context, q DBTX, tenantID uuid.UUID) (*ContactsByStatus, error)
	GetMessagesBySenderType(ctx context.Context, q DBTX, tenantID uuid.UUID) (*MessagesBySenderType, error)
	GetMessagesByDate(ctx context.Context, q DBTX, tenantID uuid.UUID, days int) ([]MessagesByDate, error)
	GetCampaignsByStatus(ctx context.Context, q DBTX, tenantID uuid.UUID) (map[string]int, error)
	GetAgentWorkload(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]AgentWorkload, error)
}
