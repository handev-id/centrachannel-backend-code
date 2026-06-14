package contact

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"encoding/csv"
	"strings"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/avatar"
	"centrachannel/internal/utils/logger"
)

type ContactService interface {
	List(ctx context.Context, q ListContactQuery, t *tenant.Tenant) (*PaginatedResponse, error)
	GetByID(ctx context.Context, tenantID int, id int) (*Contact, error)
	Create(ctx context.Context, req CreateContactRequest, t *tenant.Tenant) (*Contact, error)
	Update(ctx context.Context, tenantID int, id int, req UpdateContactRequest) (*Contact, error)
	Delete(ctx context.Context, tenantID int, id int) error
	Merge(ctx context.Context, tenantID int, sourceID int, targetID int) error
	Unmerge(ctx context.Context, tenantID int, id int) error
	GetConversations(ctx context.Context, tenantID int, contactID int) (interface{}, error)
	ImportCSV(ctx context.Context, tenantID int, records [][]string) (*CSVImportResult, error)
	ExportCSV(ctx context.Context, tenantID int) (string, error)
}

type contactService struct {
	repo   ContactRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewContactService(repo ContactRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) ContactService {
	return &contactService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *contactService) List(ctx context.Context, q ListContactQuery, t *tenant.Tenant) (*PaginatedResponse, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}
	offset := (q.Page - 1) * q.Limit

	contacts, total, err := s.repo.List(ctx, s.db, t.ID, q.Limit, offset, q.Search, q.Status, q.ChannelID)
	if err != nil {
		return nil, err
	}

	lastPage := int(math.Ceil(float64(total) / float64(q.Limit)))
	from := offset + 1
	to := offset + len(contacts)
	if to > total {
		to = total
	}
	if total == 0 {
		from = 0
		to = 0
	}

	meta := PaginationMeta{
		Total: total, PerPage: q.Limit, CurrentPage: q.Page,
		LastPage: lastPage, From: from, To: to,
	}
	return &PaginatedResponse{Meta: meta, Data: contacts}, nil
}

func (s *contactService) GetByID(ctx context.Context, tenantID int, id int) (*Contact, error) {
	return s.repo.GetByID(ctx, s.db, tenantID, id)
}

func (s *contactService) Create(ctx context.Context, req CreateContactRequest, t *tenant.Tenant) (*Contact, error) {
	status := "individual"
	if req.Status != nil {
		status = *req.Status
	}

	var dob *time.Time
	if req.DateOfBirth != nil {
		t, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err == nil {
			dob = &t
		}
	}

	avatarData := req.Avatar
	if len(avatarData) == 0 {
		avatarData = avatar.GenerateInitials(req.FirstName)
	}

	contact := &Contact{
		TenantID:  t.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		Avatar:    avatarData,
		Country:   req.Country,
		Bio:       req.Bio,
		Occupation: req.Occupation,
		Category:  req.Category,
		CategoryDescription: req.CategoryDescription,
		Gender:    req.Gender,
		DateOfBirth: dob,
		ProvinceOfOrigin: req.ProvinceOfOrigin,
		Facebook:  req.Facebook,
		Instagram: req.Instagram,
		Whatsapp:  req.Whatsapp,
		X:         req.X,
		Tiktok:    req.Tiktok,
		Status:    status,
		InstitutionName: req.InstitutionName,
	}

	id, err := s.repo.Create(ctx, s.db, contact)
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	contact.ID = id
	contact.CreatedAt = time.Now()
	contact.UpdatedAt = time.Now()
	return contact, nil
}

func (s *contactService) Update(ctx context.Context, tenantID int, id int, req UpdateContactRequest) (*Contact, error) {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.FirstName != nil { existing.FirstName = *req.FirstName }
	if req.LastName != nil { existing.LastName = req.LastName }
	if req.Username != nil { existing.Username = req.Username }
	if req.Email != nil { existing.Email = req.Email }
	if req.Phone != nil { existing.Phone = req.Phone }
	if len(req.Avatar) > 0 { existing.Avatar = req.Avatar }
	if req.Country != nil { existing.Country = req.Country }
	if req.Bio != nil { existing.Bio = req.Bio }
	if req.Occupation != nil { existing.Occupation = req.Occupation }
	if req.Category != nil { existing.Category = req.Category }
	if req.CategoryDescription != nil { existing.CategoryDescription = req.CategoryDescription }
	if req.Gender != nil { existing.Gender = req.Gender }
	if req.DateOfBirth != nil {
		if t, err := time.Parse("2006-01-02", *req.DateOfBirth); err == nil {
			existing.DateOfBirth = &t
		}
	}
	if req.ProvinceOfOrigin != nil { existing.ProvinceOfOrigin = req.ProvinceOfOrigin }
	if req.Facebook != nil { existing.Facebook = req.Facebook }
	if req.Instagram != nil { existing.Instagram = req.Instagram }
	if req.Whatsapp != nil { existing.Whatsapp = req.Whatsapp }
	if req.X != nil { existing.X = req.X }
	if req.Tiktok != nil { existing.Tiktok = req.Tiktok }
	if req.Status != nil { existing.Status = *req.Status }
	if req.InstitutionName != nil { existing.InstitutionName = req.InstitutionName }

	if err := s.repo.Update(ctx, s.db, tenantID, id, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *contactService) Delete(ctx context.Context, tenantID int, id int) error {
	return s.repo.SoftDelete(ctx, s.db, tenantID, id)
}

func (s *contactService) GetConversations(ctx context.Context, tenantID int, contactID int) (interface{}, error) {
	type convBrief struct {
		ID          int              `json:"id"`
		Status      string           `json:"status"`
		ProfileID   int              `json:"profile_id"`
		AgentID     *int             `json:"agent_id,omitempty"`
		ChannelID   int              `json:"channel_id"`
		UnreadCount int              `json:"unread_count"`
		LastMessage json.RawMessage  `json:"last_message,omitempty"`
		LastActivity *time.Time      `json:"last_activity,omitempty"`
		CreatedAt   time.Time        `json:"created_at"`
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.status, c.profile_id, c.agent_id, c.channel_id,
		       c.unread_count, c.last_message, c.last_activity, c.created_at
		FROM conversations c
		INNER JOIN profiles p ON p.id = c.profile_id
		WHERE p.contact_id = $1 AND c.tenant_id = $2
		ORDER BY c.last_activity DESC NULLS LAST
	`, contactID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []convBrief
	for rows.Next() {
		var conv convBrief
		var agentID sql.NullInt64
		var lastMessage sql.NullString
		var lastActivity sql.NullTime

		if err := rows.Scan(&conv.ID, &conv.Status, &conv.ProfileID, &agentID, &conv.ChannelID, &conv.UnreadCount, &lastMessage, &lastActivity, &conv.CreatedAt); err != nil {
			return nil, err
		}
		if agentID.Valid { id := int(agentID.Int64); conv.AgentID = &id }
		if lastMessage.Valid { conv.LastMessage = []byte(lastMessage.String) }
		if lastActivity.Valid { conv.LastActivity = &lastActivity.Time }

		convs = append(convs, conv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return convs, nil
}

func (s *contactService) Merge(ctx context.Context, tenantID int, sourceID int, targetID int) error {
	if sourceID == targetID {
		return fmt.Errorf("cannot merge contact into itself")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE contacts SET merged_to_id = $1, deleted_at = $2, updated_at = $2 WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL`, targetID, time.Now(), sourceID, tenantID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET merged_from_contact_id = $1 WHERE contact_id = $2`, sourceID, targetID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET contact_id = $1 WHERE contact_id = $2 AND deleted_at IS NULL`, targetID, sourceID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *contactService) Unmerge(ctx context.Context, tenantID int, id int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE contacts SET merged_to_id = NULL, deleted_at = NULL, updated_at = $1 WHERE id = $2 AND tenant_id = $3`, time.Now(), id, tenantID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET merged_from_contact_id = NULL WHERE merged_from_contact_id = $1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *contactService) ImportCSV(ctx context.Context, tenantID int, records [][]string) (*CSVImportResult, error) {
	if len(records) < 2 {
		return &CSVImportResult{Total: 0, Success: 0, Failed: 0}, nil
	}

	headers := records[0]
	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[h] = i
	}

	result := &CSVImportResult{Total: len(records) - 1}

	for rowIdx, row := range records[1:] {
		firstName := getCol(row, colMap, "first_name")
		if firstName == "" {
			result.Failed++
			result.Errors = append(result.Errors, CSVImportError{Row: rowIdx + 2, Field: "first_name", Error: "required"})
			continue
		}

		contact := &Contact{
			TenantID:        tenantID,
			FirstName:       firstName,
			LastName:        optionalStr(getCol(row, colMap, "last_name")),
			Username:        optionalStr(getCol(row, colMap, "username")),
			Email:           optionalStr(getCol(row, colMap, "email")),
			Phone:           optionalStr(getCol(row, colMap, "phone")),
			Facebook:        optionalStr(getCol(row, colMap, "facebook")),
			Instagram:       optionalStr(getCol(row, colMap, "instagram")),
			Whatsapp:        optionalStr(getCol(row, colMap, "whatsapp")),
			X:               optionalStr(getCol(row, colMap, "x")),
			Tiktok:          optionalStr(getCol(row, colMap, "tiktok")),
			InstitutionName: optionalStr(getCol(row, colMap, "institution_name")),
			Status:          "active",
		}

		if _, err := s.repo.Create(ctx, s.db, contact); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, CSVImportError{Row: rowIdx + 2, Field: "general", Error: err.Error()})
			continue
		}
		result.Success++
	}

	return result, nil
}

func (s *contactService) ExportCSV(ctx context.Context, tenantID int) (string, error) {
	contacts, _, err := s.repo.List(ctx, s.db, tenantID, 0, 0, "", "", 0)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	writer.Write([]string{"first_name", "last_name", "username", "email", "phone", "country", "status", "institution_name", "facebook", "instagram", "whatsapp", "x", "tiktok"})

	for _, c := range contacts {
		lastName := ""
		if c.LastName != nil {
			lastName = *c.LastName
		}
		username := ""
		if c.Username != nil {
			username = *c.Username
		}
		email := ""
		if c.Email != nil {
			email = *c.Email
		}
		phone := ""
		if c.Phone != nil {
			phone = *c.Phone
		}
		country := ""
		if c.Country != nil {
			country = *c.Country
		}
		institution := ""
		if c.InstitutionName != nil {
			institution = *c.InstitutionName
		}
		fb := ""
		if c.Facebook != nil {
			fb = *c.Facebook
		}
		ig := ""
		if c.Instagram != nil {
			ig = *c.Instagram
		}
		wa := ""
		if c.Whatsapp != nil {
			wa = *c.Whatsapp
		}
		x := ""
		if c.X != nil {
			x = *c.X
		}
		tiktok := ""
		if c.Tiktok != nil {
			tiktok = *c.Tiktok
		}

		writer.Write([]string{c.FirstName, lastName, username, email, phone, country, c.Status, institution, fb, ig, wa, x, tiktok})
	}
	writer.Flush()
	return buf.String(), nil
}

func getCol(row []string, colMap map[string]int, name string) string {
	if idx, ok := colMap[name]; ok && idx < len(row) {
		return row[idx]
	}
	return ""
}

func optionalStr(v string) *string {
	if v == "" { return nil }
	return &v
}
