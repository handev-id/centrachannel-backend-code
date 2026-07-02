package whatsapp_device

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type WhatsAppDeviceService interface {
	List(ctx context.Context, tenantID int) ([]WhatsAppDevice, error)
	GetByID(ctx context.Context, tenantID int, id int) (*WhatsAppDevice, error)
	Create(ctx context.Context, req CreateDeviceRequest, tenantID int) (*WhatsAppDevice, error)
	Update(ctx context.Context, tenantID int, id int, req UpdateDeviceRequest) (*WhatsAppDevice, error)
	Delete(ctx context.Context, tenantID int, id int) error
	Connect(ctx context.Context, tenantID int, id int) (*WhatsAppDevice, error)
	Disconnect(ctx context.Context, tenantID int, id int) (*WhatsAppDevice, error)
	Scan(ctx context.Context, tenantID int, id int) (string, error)
	SendMessage(ctx context.Context, tenantID int, deviceID int, to string, text string) (*MessageResult, error)
}

type whatsAppDeviceService struct {
	repo   WhatsAppDeviceRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
	client WhatsAppClient
}

func NewWhatsAppDeviceService(repo WhatsAppDeviceRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, client ...WhatsAppClient) WhatsAppDeviceService {
	svc := &whatsAppDeviceService{repo: repo, db: db, cfg: cfg, logger: logger}
	if len(client) > 0 {
		svc.client = client[0]
	}
	return svc
}

func (s *whatsAppDeviceService) List(ctx context.Context, tenantID int) ([]WhatsAppDevice, error) {
	return s.repo.List(ctx, s.db, tenantID)
}

func (s *whatsAppDeviceService) GetByID(ctx context.Context, tenantID int, id int) (*WhatsAppDevice, error) {
	return s.repo.GetByID(ctx, s.db, tenantID, id)
}

func (s *whatsAppDeviceService) Create(ctx context.Context, req CreateDeviceRequest, tenantID int) (*WhatsAppDevice, error) {
	device := &WhatsAppDevice{
		TenantID:    tenantID,
		Name:        req.Name,
		CountryCode: req.CountryCode,
		Phone:       req.Phone,
		WhatsappID:  generateWhatsappID(req.Name, req.Phone),
		Status:      "DISCONNECTED",
	}
	id, err := s.repo.Create(ctx, s.db, device)
	if err != nil {
		return nil, err
	}
	device.ID = id
	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()

	if s.client != nil && s.cfg.EvolutionAPIURL != "" {
		if err := s.client.CreateInstance(ctx, device); err != nil {
			s.logger.Warn("failed to create evolution instance: %v", err)
		}
	}

	return device, nil
}

func (s *whatsAppDeviceService) Update(ctx context.Context, tenantID int, id int, req UpdateDeviceRequest) (*WhatsAppDevice, error) {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil { existing.Name = *req.Name }
	if req.CountryCode != nil { existing.CountryCode = *req.CountryCode }
	if req.Phone != nil { existing.Phone = *req.Phone }
	if req.Status != nil { existing.Status = *req.Status }

	if err := s.repo.Update(ctx, s.db, tenantID, id, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *whatsAppDeviceService) Delete(ctx context.Context, tenantID int, id int) error {
	device, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return err
	}

	if s.client != nil && s.cfg.EvolutionAPIURL != "" {
		if err := s.client.DeleteInstance(ctx, device); err != nil {
			s.logger.Warn("failed to delete evolution instance: %v", err)
		}
	}

	return s.repo.Delete(ctx, s.db, tenantID, id)
}

func (s *whatsAppDeviceService) Connect(ctx context.Context, tenantID int, id int) (*WhatsAppDevice, error) {
	device, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	if device.WhatsappID == "" {
		return nil, fmt.Errorf("device has no whatsapp_id")
	}
	if s.client != nil {
		if _, err := s.client.CheckConnection(ctx, device); err != nil {
			return nil, err
		}
		if s.cfg.EvolutionAPIURL != "" && s.cfg.WebhookBaseURL != "" {
			webhookURL := strings.TrimRight(s.cfg.WebhookBaseURL, "/") + "/webhook/evolution"
			if err := s.client.SetWebhook(ctx, device, webhookURL); err != nil {
				s.logger.Warn("failed to set webhook: %v", err)
			}
		}
	}
	device.Status = "CONNECTED"
	if err := s.repo.Update(ctx, s.db, tenantID, id, device); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *whatsAppDeviceService) Disconnect(ctx context.Context, tenantID int, id int) (*WhatsAppDevice, error) {
	device, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	if s.client != nil {
		if err := s.client.Disconnect(ctx, device); err != nil {
			return nil, err
		}
	}
	device.Status = "DISCONNECTED"
	if err := s.repo.Update(ctx, s.db, tenantID, id, device); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *whatsAppDeviceService) Scan(ctx context.Context, tenantID int, id int) (string, error) {
	device, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return "", err
	}
	if s.client != nil && s.cfg.EvolutionAPIURL != "" {
		return s.client.GetQR(ctx, device)
	}
	return fmt.Sprintf("mock_qr_%d", time.Now().UnixMilli()), nil
}

func (s *whatsAppDeviceService) SendMessage(ctx context.Context, tenantID int, deviceID int, to string, text string) (*MessageResult, error) {
	device, err := s.repo.GetByID(ctx, s.db, tenantID, deviceID)
	if err != nil {
		return nil, err
	}
	if s.client == nil {
		return nil, fmt.Errorf("whatsapp client not configured")
	}
	return s.client.SendMessage(ctx, device, to, text)
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func generateWhatsappID(name, phone string) string {
	slug := strings.ToLower(name)
	slug = nonAlphaNum.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	cleanPhone := nonAlphaNum.ReplaceAllString(phone, "")
	return fmt.Sprintf("%s-%s", slug, cleanPhone)
}
