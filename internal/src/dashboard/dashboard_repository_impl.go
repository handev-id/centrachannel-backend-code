package dashboard

import (
	"context"
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

func (r *dashboardRepository) GetActiveConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND status != 'resolved'`, tenantID).Scan(&count)
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
