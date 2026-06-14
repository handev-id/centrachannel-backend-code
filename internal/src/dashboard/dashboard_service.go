package dashboard

import (
	"context"
	"database/sql"
)

type DashboardService interface {
	GetStats(ctx context.Context, tenantID int) (*StatsResponse, error)
	GetChart(ctx context.Context, tenantID int, days int) (*ChartResponse, error)
}

type dashboardService struct {
	repo DashboardRepository
	db   *sql.DB
}

func NewDashboardService(repo DashboardRepository, db *sql.DB) DashboardService {
	return &dashboardService{repo: repo, db: db}
}

func (s *dashboardService) GetStats(ctx context.Context, tenantID int) (*StatsResponse, error) {
	totalContacts, err := s.repo.GetTotalContacts(ctx, s.db, tenantID)
	if err != nil {
		return nil, err
	}

	activeConvs, err := s.repo.GetActiveConversations(ctx, s.db, tenantID)
	if err != nil {
		return nil, err
	}

	resolvedToday, err := s.repo.GetResolvedToday(ctx, s.db, tenantID)
	if err != nil {
		return nil, err
	}

	unassigned, err := s.repo.GetUnassignedCount(ctx, s.db, tenantID)
	if err != nil {
		return nil, err
	}

	return &StatsResponse{
		TotalContacts:       totalContacts,
		ActiveConversations: activeConvs,
		ResolvedToday:       resolvedToday,
		UnassignedCount:     unassigned,
	}, nil
}

func (s *dashboardService) GetChart(ctx context.Context, tenantID int, days int) (*ChartResponse, error) {
	if days < 1 || days > 365 {
		days = 30
	}

	data, err := s.repo.GetConversationChart(ctx, s.db, tenantID, days)
	if err != nil {
		return nil, err
	}

	return &ChartResponse{Data: data}, nil
}
