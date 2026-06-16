package dashboard

import (
	"context"
	"database/sql"
)

type DashboardService interface {
	GetStats(ctx context.Context, tenantID int, days int) (*StatsResponse, error)
	GetChart(ctx context.Context, tenantID int, days int) (*ChartResponse, error)
}

type dashboardService struct {
	repo DashboardRepository
	db   *sql.DB
}

func NewDashboardService(repo DashboardRepository, db *sql.DB) DashboardService {
	return &dashboardService{repo: repo, db: db}
}

func (s *dashboardService) GetStats(ctx context.Context, tenantID int, days int) (*StatsResponse, error) {
	if days < 1 || days > 365 {
		days = 30
	}

	var (
		totalContacts,
		newContacts,
		totalConversations,
		activeConversations,
		newConversations,
		resolvedToday,
		totalUnreadConvs,
		totalUnreadMsgs,
		totalMessages,
		messagesToday,
		totalCampaigns,
		connectedDevices int
		convByStatus      *ConversationsByStatus
		convByChannel     []ConversationsByChannel
		recentConvs       []RecentConversation
		convTrend         []ChartDataPoint
		contactsByStatus  *ContactsByStatus
		msgsBySender      *MessagesBySenderType
		msgsByDate        []MessagesByDate
		campaignsByStatus map[string]int
		agentWorkload     []AgentWorkload
		err               error
	)

	if totalContacts, err = s.repo.GetTotalContacts(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if newContacts, err = s.repo.GetNewContacts(ctx, s.db, tenantID, days); err != nil {
		return nil, err
	}
	if totalConversations, err = s.repo.GetTotalConversations(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if activeConversations, err = s.repo.GetActiveConversations(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if newConversations, err = s.repo.GetNewConversations(ctx, s.db, tenantID, days); err != nil {
		return nil, err
	}
	if resolvedToday, err = s.repo.GetResolvedToday(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if totalUnreadConvs, err = s.repo.GetTotalUnreadConversations(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if totalUnreadMsgs, err = s.repo.GetTotalUnreadMessages(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if totalMessages, err = s.repo.GetTotalMessages(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if messagesToday, err = s.repo.GetMessagesToday(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if totalCampaigns, err = s.repo.GetTotalCampaigns(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if connectedDevices, err = s.repo.GetConnectedWhatsappDevices(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if convByStatus, err = s.repo.GetConversationsByStatus(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if convByChannel, err = s.repo.GetConversationsByChannel(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if recentConvs, err = s.repo.GetRecentConversations(ctx, s.db, tenantID, 10); err != nil {
		return nil, err
	}
	if convTrend, err = s.repo.GetConversationChart(ctx, s.db, tenantID, days); err != nil {
		return nil, err
	}
	if contactsByStatus, err = s.repo.GetContactsByStatus(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if msgsBySender, err = s.repo.GetMessagesBySenderType(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if msgsByDate, err = s.repo.GetMessagesByDate(ctx, s.db, tenantID, days); err != nil {
		return nil, err
	}
	if campaignsByStatus, err = s.repo.GetCampaignsByStatus(ctx, s.db, tenantID); err != nil {
		return nil, err
	}
	if agentWorkload, err = s.repo.GetAgentWorkload(ctx, s.db, tenantID); err != nil {
		return nil, err
	}

	return &StatsResponse{
		Summary: SummaryResponse{
			TotalContacts:            totalContacts,
			NewContacts:              newContacts,
			TotalConversations:       totalConversations,
			ActiveConversations:      activeConversations,
			NewConversations:         newConversations,
			ResolvedToday:            resolvedToday,
			TotalUnreadConversations: totalUnreadConvs,
			TotalUnreadMessages:      totalUnreadMsgs,
			TotalMessages:            totalMessages,
			MessagesToday:            messagesToday,
			TotalCampaigns:           totalCampaigns,
			ConnectedWhatsappDevices: connectedDevices,
		},
		Conversations: ConversationsResponse{
			ByStatus:  *convByStatus,
			ByChannel: convByChannel,
			Recent:    recentConvs,
			Trend:     convTrend,
		},
		Contacts: ContactsResponse{
			ByStatus:    *contactsByStatus,
			NewContacts: newContacts,
		},
		Messages: MessagesResponse{
			Total:        totalMessages,
			Today:        messagesToday,
			BySenderType: *msgsBySender,
			ByDate:       msgsByDate,
		},
		Campaigns: CampaignsResponse{
			Total:    totalCampaigns,
			ByStatus: campaignsByStatus,
		},
		Agents: agentWorkload,
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
