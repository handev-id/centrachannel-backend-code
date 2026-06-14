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
	GetActiveConversations(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetResolvedToday(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetUnassignedCount(ctx context.Context, q DBTX, tenantID int) (int, error)
	GetConversationChart(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error)
}
