package campaign

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

type CampaignService interface {
	List(ctx context.Context, q ListCampaignQuery, t *tenant.Tenant) (*PaginatedResponse, error)
	GetByID(ctx context.Context, tenantID int, id int) (*Campaign, error)
	Create(ctx context.Context, req CreateCampaignRequest, t *tenant.Tenant, userID int) (*Campaign, error)
	Update(ctx context.Context, tenantID int, id int, req UpdateCampaignRequest) (*Campaign, error)
	Delete(ctx context.Context, tenantID int, id int) error
	Send(ctx context.Context, tenantID int, id int) error
}

type PaginationMeta struct {
	Total       int `json:"total"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	From        int `json:"from"`
	To          int `json:"to"`
}

type PaginatedResponse struct {
	Meta PaginationMeta `json:"meta"`
	Data interface{}    `json:"data"`
}

type ListCampaignQuery struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

type campaignService struct {
	repo   CampaignRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewCampaignService(repo CampaignRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) CampaignService {
	return &campaignService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *campaignService) List(ctx context.Context, q ListCampaignQuery, t *tenant.Tenant) (*PaginatedResponse, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	offset := (q.Page - 1) * q.Limit

	campaigns, total, err := s.repo.List(ctx, s.db, t.ID, q.Limit, offset, q.Search)
	if err != nil {
		s.logger.Error("Failed to list campaigns: %v", err)
		return nil, fmt.Errorf("failed to list campaigns: %w", err)
	}

	lastPage := int(math.Ceil(float64(total) / float64(q.Limit)))
	if lastPage < 1 {
		lastPage = 1
	}

	from := offset + 1
	to := offset + len(campaigns)
	if total == 0 {
		from = 0
		to = 0
	}

	return &PaginatedResponse{
		Meta: PaginationMeta{
			Total:       total,
			PerPage:     q.Limit,
			CurrentPage: q.Page,
			LastPage:    lastPage,
			From:        from,
			To:          to,
		},
		Data: campaigns,
	}, nil
}

func (s *campaignService) GetByID(ctx context.Context, tenantID int, id int) (*Campaign, error) {
	campaign, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, fmt.Errorf("campaign not found")
	}
	return campaign, nil
}

func (s *campaignService) Create(ctx context.Context, req CreateCampaignRequest, t *tenant.Tenant, userID int) (*Campaign, error) {
	var scheduledAt *time.Time
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_at format, use RFC3339")
		}
		scheduledAt = &t
	}

	campaign := &Campaign{
		TenantID:        t.ID,
		Name:            req.Name,
		Description:     req.Description,
		MessageTemplate: req.MessageTemplate,
		ChannelID:       req.ChannelID,
		Status:          "draft",
		ScheduledAt:     scheduledAt,
		TotalCount:      len(req.ContactIDs),
		CreatedBy:       userID,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	campaignID, err := s.repo.Create(ctx, tx, campaign)
	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	for _, contactID := range req.ContactIDs {
		if err := s.repo.CreateContact(ctx, tx, campaignID, contactID); err != nil {
			return nil, fmt.Errorf("failed to attach contact %d: %w", contactID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	campaign.ID = campaignID
	return campaign, nil
}

func (s *campaignService) Update(ctx context.Context, tenantID int, id int, req UpdateCampaignRequest) (*Campaign, error) {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("campaign not found")
	}

	if existing.Status != "draft" {
		return nil, fmt.Errorf("can only update campaigns in draft status")
	}

	updated := &Campaign{
		Name:            existing.Name,
		Description:     existing.Description,
		MessageTemplate: existing.MessageTemplate,
		Status:          existing.Status,
		ScheduledAt:     existing.ScheduledAt,
	}

	if req.Name != nil {
		updated.Name = *req.Name
	}
	if req.Description != nil {
		updated.Description = req.Description
	}
	if req.MessageTemplate != nil {
		updated.MessageTemplate = *req.MessageTemplate
	}
	if req.Status != nil {
		updated.Status = *req.Status
	}
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_at format, use RFC3339")
		}
		updated.ScheduledAt = &t
	}

	if err := s.repo.Update(ctx, s.db, tenantID, id, updated); err != nil {
		return nil, fmt.Errorf("failed to update campaign: %w", err)
	}

	return s.repo.GetByID(ctx, s.db, tenantID, id)
}

func (s *campaignService) Delete(ctx context.Context, tenantID int, id int) error {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("campaign not found")
	}

	if existing.Status == "sending" || existing.Status == "sent" {
		return fmt.Errorf("cannot delete campaign with status '%s'", existing.Status)
	}

	return s.repo.Delete(ctx, s.db, tenantID, id)
}

func (s *campaignService) Send(ctx context.Context, tenantID int, id int) error {
	campaign, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return err
	}
	if campaign == nil {
		return fmt.Errorf("campaign not found")
	}

	if campaign.Status != "draft" && campaign.Status != "failed" {
		return fmt.Errorf("campaign must be in draft or failed status to send")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	updateCampaign := &Campaign{Status: "sending"}
	if err := s.repo.Update(ctx, tx, tenantID, id, updateCampaign); err != nil {
		return fmt.Errorf("failed to update campaign status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	go func() {
		processCtx := context.Background()
		s.processSend(processCtx, tenantID, id)
	}()

	return nil
}

func (s *campaignService) processSend(ctx context.Context, tenantID int, campaignID int) {
	batchSize := 50
	sentCount := 0

	for {
		contacts, err := s.repo.GetPendingContacts(ctx, s.db, campaignID, batchSize)
		if err != nil {
			s.logger.Error("Failed to get pending contacts for campaign %d: %v", campaignID, err)
			s.repo.Update(ctx, s.db, tenantID, campaignID, &Campaign{Status: "failed"})
			return
		}

		if len(contacts) == 0 {
			break
		}

		for _, cc := range contacts {
			err := s.repo.UpdateContactStatus(ctx, s.db, campaignID, cc.ContactID, "sent", nil)
			if err != nil {
				s.logger.Error("Failed to update contact %d status: %v", cc.ContactID, err)
				errMsg := err.Error()
				s.repo.UpdateContactStatus(ctx, s.db, campaignID, cc.ContactID, "failed", &errMsg)
				continue
			}
			sentCount++
		}
	}

	finalStatus := "sent"
	remaining, err := s.repo.CountPendingContacts(ctx, s.db, campaignID)
	if err == nil && remaining > 0 {
		finalStatus = "partial"
	}

	s.repo.Update(ctx, s.db, tenantID, campaignID, &Campaign{
		Status:    finalStatus,
		SentCount: sentCount,
	})
	s.logger.Info("Campaign %d processed: %d sent, status: %s", campaignID, sentCount, finalStatus)
}
