package campaign

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"centrachannel/config"
	"centrachannel/internal/messenger"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/logger"
)

type CampaignService interface {
	List(ctx context.Context, q ListCampaignQuery, t *tenant.Tenant) ([]*Campaign, int, error)
	GetByID(ctx context.Context, tenantID int, id int) (*Campaign, error)
	Create(ctx context.Context, req CreateCampaignRequest, t *tenant.Tenant, userID int) (*Campaign, error)
	Update(ctx context.Context, tenantID int, id int, req UpdateCampaignRequest) (*Campaign, error)
	Delete(ctx context.Context, tenantID int, id int) error
	Send(ctx context.Context, tenantID int, id int) error
	ListTemplates(ctx context.Context, tenantID int) ([]CampaignTemplate, error)
	GetTemplateByID(ctx context.Context, tenantID int, id int) (*CampaignTemplate, error)
	CreateTemplate(ctx context.Context, req CreateTemplateRequest, t *tenant.Tenant) (*CampaignTemplate, error)
	UpdateTemplate(ctx context.Context, tenantID int, id int, req UpdateTemplateRequest) (*CampaignTemplate, error)
	DeleteTemplate(ctx context.Context, tenantID int, id int) error
	ListRecipientLists(ctx context.Context, tenantID int) ([]CampaignRecipientList, error)
	GetRecipientListByID(ctx context.Context, tenantID int, id int) (*CampaignRecipientList, error)
	CreateRecipientList(ctx context.Context, req CreateRecipientListRequest, t *tenant.Tenant) (*CampaignRecipientList, error)
	UpdateRecipientList(ctx context.Context, tenantID int, id int, req UpdateRecipientListRequest) (*CampaignRecipientList, error)
	DeleteRecipientList(ctx context.Context, tenantID int, id int) error
	ListRecipientContacts(ctx context.Context, listID int) ([]CampaignRecipientContact, error)
	AddRecipientContact(ctx context.Context, listID int, req AddContactToListRequest) (*CampaignRecipientContact, error)
	RemoveRecipientContact(ctx context.Context, id int) error
}

type campaignService struct {
	repo       CampaignRepository
	deviceRepo whatsapp_device.WhatsAppDeviceRepository
	channelRepo channel.ChannelRepository
	db         *sql.DB
	cfg        *config.Config
	logger     *logger.Logger
}

func NewCampaignService(repo CampaignRepository, deviceRepo whatsapp_device.WhatsAppDeviceRepository, channelRepo channel.ChannelRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) CampaignService {
	return &campaignService{repo: repo, deviceRepo: deviceRepo, channelRepo: channelRepo, db: db, cfg: cfg, logger: logger}
}

func (s *campaignService) List(ctx context.Context, q ListCampaignQuery, t *tenant.Tenant) ([]*Campaign, int, error) {
	if q.Page < 1 { q.Page = 1 }
	if q.Limit < 1 || q.Limit > 100 { q.Limit = 20 }
	offset := (q.Page - 1) * q.Limit

	campaigns, total, err := s.repo.List(ctx, s.db, t.ID, q.Limit, offset, q.Search)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list campaigns: %w", err)
	}

	return campaigns, total, nil
}

func (s *campaignService) GetByID(ctx context.Context, tenantID int, id int) (*Campaign, error) {
	campaign, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil { return nil, err }
	if campaign == nil { return nil, fmt.Errorf("campaign not found") }
	return campaign, nil
}

func (s *campaignService) Create(ctx context.Context, req CreateCampaignRequest, t *tenant.Tenant, userID int) (*Campaign, error) {
	campaignType := req.Type
	if campaignType == "" { campaignType = "broadcast" }

	var scheduledAt *time.Time
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil { return nil, fmt.Errorf("invalid scheduled_at format, use RFC3339") }
		scheduledAt = &parsed
	}

	campaign := &Campaign{
		TenantID:        t.ID,
		Name:            req.Name,
		Type:            campaignType,
		Description:     req.Description,
		MessageTemplate: req.MessageTemplate,
		ChannelID:       req.ChannelID,
		Status:          "draft",
		SendingOption:   req.SendingOption,
		ScheduledAt:     scheduledAt,
		RecipientListID: req.RecipientListID,
		TemplateID:      req.TemplateID,
		CreatedBy:       userID,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return nil, fmt.Errorf("failed to begin transaction: %w", err) }
	defer tx.Rollback()

	campaignID, err := s.repo.Create(ctx, tx, campaign)
	if err != nil { return nil, fmt.Errorf("failed to create campaign: %w", err) }

	if req.RecipientListID != nil {
		contacts, err := s.repo.ListRecipientContacts(ctx, s.db, *req.RecipientListID)
		if err != nil { return nil, fmt.Errorf("failed to fetch recipient contacts: %w", err) }

		for _, c := range contacts {
			recipient := &CampaignRecipient{
				CampaignID:        campaignID,
				RecipientContactID: c.ID,
				Status:            "pending",
			}
			if _, err := s.repo.CreateRecipient(ctx, tx, recipient); err != nil {
				return nil, fmt.Errorf("failed to add recipient: %w", err)
			}
			campaign.TotalCount++
		}
	}

	if err := tx.Commit(); err != nil { return nil, fmt.Errorf("failed to commit: %w", err) }

	campaign.ID = campaignID
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	return campaign, nil
}

func (s *campaignService) Update(ctx context.Context, tenantID int, id int, req UpdateCampaignRequest) (*Campaign, error) {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil { return nil, err }
	if existing == nil { return nil, fmt.Errorf("campaign not found") }
	if existing.Status != "draft" { return nil, fmt.Errorf("can only update campaigns in draft status") }

	updated := &Campaign{
		Name: existing.Name, Type: existing.Type, Description: existing.Description,
		MessageTemplate: existing.MessageTemplate, Status: existing.Status,
		SendingOption: existing.SendingOption, ScheduledAt: existing.ScheduledAt,
		RecipientListID: existing.RecipientListID, TemplateID: existing.TemplateID,
		AgentID: existing.AgentID, SenderID: existing.SenderID,
	}

	if req.Name != nil { updated.Name = *req.Name }
	if req.Type != nil { updated.Type = *req.Type }
	if req.Description != nil { updated.Description = req.Description }
	if req.MessageTemplate != nil { updated.MessageTemplate = *req.MessageTemplate }
	if req.Status != nil { updated.Status = *req.Status }
	if len(req.SendingOption) > 0 { updated.SendingOption = req.SendingOption }
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil { return nil, fmt.Errorf("invalid scheduled_at format, use RFC3339") }
		updated.ScheduledAt = &parsed
	}
	if req.RecipientListID != nil { updated.RecipientListID = req.RecipientListID }
	if req.TemplateID != nil { updated.TemplateID = req.TemplateID }
	if req.AgentID != nil { updated.AgentID = req.AgentID }
	if req.SenderID != nil { updated.SenderID = req.SenderID }

	if err := s.repo.Update(ctx, s.db, tenantID, id, updated); err != nil {
		return nil, fmt.Errorf("failed to update campaign: %w", err)
	}
	return s.repo.GetByID(ctx, s.db, tenantID, id)
}

func (s *campaignService) Delete(ctx context.Context, tenantID int, id int) error {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil { return err }
	if existing == nil { return fmt.Errorf("campaign not found") }
	if existing.Status == "sending" || existing.Status == "sent" {
		return fmt.Errorf("cannot delete campaign with status '%s'", existing.Status)
	}
	return s.repo.Delete(ctx, s.db, tenantID, id)
}

func (s *campaignService) Send(ctx context.Context, tenantID int, id int) error {
	campaign, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil { return err }
	if campaign == nil { return fmt.Errorf("campaign not found") }
	if campaign.Status != "draft" && campaign.Status != "failed" {
		return fmt.Errorf("campaign must be in draft or failed status to send")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return fmt.Errorf("failed to begin transaction: %w", err) }
	defer tx.Rollback()

	if err := s.repo.Update(ctx, tx, tenantID, id, &Campaign{Status: "sending"}); err != nil {
		return fmt.Errorf("failed to update campaign status: %w", err)
	}
	if err := tx.Commit(); err != nil { return fmt.Errorf("failed to commit: %w", err) }

	go func() {
		s.processSend(context.Background(), tenantID, id)
	}()

	return nil
}

func (s *campaignService) loadTemplateText(campaign *Campaign) (string, error) {
	if campaign.TemplateID == nil {
		return campaign.MessageTemplate, nil
	}

	tmpl, err := s.repo.GetTemplateByID(context.Background(), s.db, campaign.TenantID, *campaign.TemplateID)
	if err != nil || tmpl == nil {
		s.logger.Warn("Template %d not found for campaign %d, falling back to message_template", *campaign.TemplateID, campaign.ID)
		return campaign.MessageTemplate, nil
	}

	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(tmpl.Content, &content); err != nil || content.Text == "" {
		return campaign.MessageTemplate, nil
	}

	return content.Text, nil
}

func (s *campaignService) processSend(ctx context.Context, tenantID int, campaignID int) {
	campaign, err := s.repo.GetByID(ctx, s.db, tenantID, campaignID)
	if err != nil {
		s.logger.Error("Failed to get campaign %d: %v", campaignID, err)
		return
	}

	templateText, err := s.loadTemplateText(campaign)
	if err != nil {
		s.logger.Error("Failed to load template for campaign %d: %v", campaignID, err)
		s.repo.Update(ctx, s.db, tenantID, campaignID, &Campaign{Status: "failed"})
		return
	}

	ch, err := s.channelRepo.GetByID(ctx, s.db, campaign.ChannelID)
	if err != nil {
		s.logger.Error("Failed to get channel for campaign %d: %v", campaignID, err)
		s.repo.Update(ctx, s.db, tenantID, campaignID, &Campaign{Status: "failed"})
		return
	}

	var device *whatsapp_device.WhatsAppDevice
	if campaign.SenderID != nil && s.cfg.EvolutionAPIURL != "" {
		device, err = s.deviceRepo.GetByID(ctx, s.db, tenantID, *campaign.SenderID)
		if err != nil {
			s.logger.Error("Failed to get sender device for campaign %d: %v", campaignID, err)
		}
	}

	batchSize := 50
	sentCount := 0

	for {
		recipients, err := s.repo.GetPendingRecipientsWithPhone(ctx, s.db, campaignID, batchSize)
		if err != nil {
			s.logger.Error("Failed to get pending recipients for campaign %d: %v", campaignID, err)
			s.repo.Update(ctx, s.db, tenantID, campaignID, &Campaign{Status: "failed"})
			return
		}
		if len(recipients) == 0 { break }

		for _, r := range recipients {
			now := time.Now()
			status := "delivered"
			var failedReason *string

			if device != nil && (ch.Type == "whatsapp" || ch.Type == "whatsapp_business") && r.Phone != "" {
				evoCfg := messenger.EvolutionConfig{
					APIURL:   s.cfg.EvolutionAPIURL,
					APIKey:   s.cfg.EvolutionAPIKey,
					DeviceID: device.WhatsappID,
				}
				sender := messenger.NewEvolutionSender(evoCfg, s.logger)
				recipientID := r.Phone
				if !strings.HasPrefix(recipientID, "+") && !strings.HasPrefix(recipientID, "55") {
					recipientID = device.CountryCode + r.Phone
				}

				text := templateText
				if r.Message != "" {
					text = r.Message
				}
				firstName := r.FirstName
				if firstName == "" {
					firstName = r.Phone
				}
				text = strings.NewReplacer(
					"{{first_name}}", firstName,
					"{{phone}}", r.Phone,
				).Replace(text)

				outMsg := &messenger.OutgoingMessage{
					ChannelType: ch.Type,
					RecipientID: recipientID,
					Text:        &text,
				}

				if _, err := sender.Send(outMsg); err != nil {
					s.logger.Error("Failed to send campaign message to %s: %v", r.Phone, err)
					status = "failed"
					errStr := err.Error()
					failedReason = &errStr
				}
			}

			if err := s.repo.UpdateRecipientStatus(ctx, s.db, r.ID, status, failedReason, &sql.NullTime{Time: now, Valid: true}); err != nil {
				s.logger.Error("Failed to update recipient %d: %v", r.ID, err)
			}
			if status == "delivered" {
				sentCount++
			}

			time.Sleep(1200 * time.Millisecond)
		}
	}

	finalStatus := "sent"
	remaining, err := s.repo.CountPendingRecipients(ctx, s.db, campaignID)
	if err == nil && remaining > 0 { finalStatus = "partial" }
	if sentCount == 0 { finalStatus = "failed" }

	s.repo.Update(ctx, s.db, tenantID, campaignID, &Campaign{
		Status:    finalStatus,
		SentCount: sentCount,
	})
	s.logger.Info("Campaign %d processed: %d sent, status: %s", campaignID, sentCount, finalStatus)
}

func (s *campaignService) ListTemplates(ctx context.Context, tenantID int) ([]CampaignTemplate, error) {
	return s.repo.ListTemplates(ctx, s.db, tenantID)
}

func (s *campaignService) GetTemplateByID(ctx context.Context, tenantID int, id int) (*CampaignTemplate, error) {
	t, err := s.repo.GetTemplateByID(ctx, s.db, tenantID, id)
	if err != nil { return nil, err }
	if t == nil { return nil, fmt.Errorf("template not found") }
	return t, nil
}

func (s *campaignService) CreateTemplate(ctx context.Context, req CreateTemplateRequest, t *tenant.Tenant) (*CampaignTemplate, error) {
	variables := req.Variables
	if len(variables) == 0 { variables = []byte("[]") }

	template := &CampaignTemplate{
		TenantID:     t.ID,
		Name:         req.Name,
		Type:         req.Type,
		TemplateType: req.TemplateType,
		Category:     req.Category,
		Language:     req.Language,
		Content:      req.Content,
		Variables:    variables,
		Quality:      req.Quality,
		AccountID:    req.AccountID,
	}

	id, err := s.repo.CreateTemplate(ctx, s.db, template)
	if err != nil { return nil, fmt.Errorf("failed to create template: %w", err) }

	template.ID = id
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	return template, nil
}

func (s *campaignService) UpdateTemplate(ctx context.Context, tenantID int, id int, req UpdateTemplateRequest) (*CampaignTemplate, error) {
	existing, err := s.repo.GetTemplateByID(ctx, s.db, tenantID, id)
	if err != nil { return nil, err }
	if existing == nil { return nil, fmt.Errorf("template not found") }

	updated := &CampaignTemplate{
		Name: existing.Name, Type: existing.Type, TemplateType: existing.TemplateType,
		Category: existing.Category, Language: existing.Language, Content: existing.Content,
		Variables: existing.Variables, Quality: existing.Quality, AccountID: existing.AccountID,
	}

	if req.Name != nil { updated.Name = *req.Name }
	if req.Type != nil { updated.Type = req.Type }
	if req.TemplateType != nil { updated.TemplateType = req.TemplateType }
	if req.Category != nil { updated.Category = req.Category }
	if req.Language != nil { updated.Language = req.Language }
	if len(req.Content) > 0 { updated.Content = req.Content }
	if len(req.Variables) > 0 { updated.Variables = req.Variables }
	if req.Quality != nil { updated.Quality = req.Quality }
	if req.AccountID != nil { updated.AccountID = req.AccountID }

	if err := s.repo.UpdateTemplate(ctx, s.db, tenantID, id, updated); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}
	return s.repo.GetTemplateByID(ctx, s.db, tenantID, id)
}

func (s *campaignService) DeleteTemplate(ctx context.Context, tenantID int, id int) error {
	return s.repo.DeleteTemplate(ctx, s.db, tenantID, id)
}

func (s *campaignService) ListRecipientLists(ctx context.Context, tenantID int) ([]CampaignRecipientList, error) {
	return s.repo.ListRecipientLists(ctx, s.db, tenantID)
}

func (s *campaignService) GetRecipientListByID(ctx context.Context, tenantID int, id int) (*CampaignRecipientList, error) {
	l, err := s.repo.GetRecipientListByID(ctx, s.db, tenantID, id)
	if err != nil { return nil, err }
	if l == nil { return nil, fmt.Errorf("recipient list not found") }
	return l, nil
}

func (s *campaignService) CreateRecipientList(ctx context.Context, req CreateRecipientListRequest, t *tenant.Tenant) (*CampaignRecipientList, error) {
	source := req.Source
	if source == "" { source = "manual" }

	list := &CampaignRecipientList{
		TenantID:  t.ID,
		Name:      req.Name,
		Source:    source,
		ChannelID: req.ChannelID,
	}

	id, err := s.repo.CreateRecipientList(ctx, s.db, list)
	if err != nil { return nil, fmt.Errorf("failed to create recipient list: %w", err) }

	list.ID = id
	list.CreatedAt = time.Now()
	list.UpdatedAt = time.Now()
	return list, nil
}

func (s *campaignService) UpdateRecipientList(ctx context.Context, tenantID int, id int, req UpdateRecipientListRequest) (*CampaignRecipientList, error) {
	existing, err := s.repo.GetRecipientListByID(ctx, s.db, tenantID, id)
	if err != nil { return nil, err }
	if existing == nil { return nil, fmt.Errorf("recipient list not found") }

	updated := &CampaignRecipientList{
		Name: existing.Name, Source: existing.Source, Status: existing.Status, ChannelID: existing.ChannelID,
	}

	if req.Name != nil { updated.Name = *req.Name }
	if req.Source != nil { updated.Source = *req.Source }
	if req.Status != nil { updated.Status = req.Status }
	if req.ChannelID != nil { updated.ChannelID = req.ChannelID }

	if err := s.repo.UpdateRecipientList(ctx, s.db, tenantID, id, updated); err != nil {
		return nil, fmt.Errorf("failed to update recipient list: %w", err)
	}
	return s.repo.GetRecipientListByID(ctx, s.db, tenantID, id)
}

func (s *campaignService) DeleteRecipientList(ctx context.Context, tenantID int, id int) error {
	return s.repo.DeleteRecipientList(ctx, s.db, tenantID, id)
}

func (s *campaignService) ListRecipientContacts(ctx context.Context, listID int) ([]CampaignRecipientContact, error) {
	return s.repo.ListRecipientContacts(ctx, s.db, listID)
}

func (s *campaignService) AddRecipientContact(ctx context.Context, listID int, req AddContactToListRequest) (*CampaignRecipientContact, error) {
	contact := &CampaignRecipientContact{
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		Username:              req.Username,
		Institution:           req.Institution,
		Email:                 req.Email,
		Phone:                 req.Phone,
		CampaignRecipientListID: listID,
		MasterContactID:       req.MasterContactID,
	}

	id, err := s.repo.CreateRecipientContact(ctx, s.db, contact)
	if err != nil { return nil, fmt.Errorf("failed to add contact: %w", err) }

	contact.ID = id
	contact.CreatedAt = time.Now()
	contact.UpdatedAt = time.Now()
	return contact, nil
}

func (s *campaignService) RemoveRecipientContact(ctx context.Context, id int) error {
	return s.repo.DeleteRecipientContact(ctx, s.db, id)
}
