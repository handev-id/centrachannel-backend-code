package contact

import (
	"encoding/json"
	"time"

	"centrachannel/internal/src/profile"
	"centrachannel/internal/utils"
)

type Contact struct {
	ID                  int                `json:"id"`
	TenantID            int                `json:"tenant_id"`
	FirstName           string             `json:"first_name"`
	LastName            *string            `json:"last_name,omitempty"`
	Username            *string            `json:"username,omitempty"`
	Email               *string            `json:"email,omitempty"`
	Phone               *string            `json:"phone,omitempty"`
	Avatar              json.RawMessage    `json:"avatar,omitempty"`
	Country             *string            `json:"country,omitempty"`
	Bio                 *string            `json:"bio,omitempty"`
	Occupation          *string            `json:"occupation,omitempty"`
	Category            *string            `json:"category,omitempty"`
	CategoryDescription *string            `json:"category_description,omitempty"`
	Gender              *string            `json:"gender,omitempty"`
	DateOfBirth         *time.Time         `json:"date_of_birth,omitempty"`
	ProvinceOfOrigin    *string            `json:"province_of_origin,omitempty"`
	Facebook            *string            `json:"facebook,omitempty"`
	Instagram           *string            `json:"instagram,omitempty"`
	Whatsapp            *string            `json:"whatsapp,omitempty"`
	X                   *string            `json:"x,omitempty"`
	Tiktok              *string            `json:"tiktok,omitempty"`
	Status              string             `json:"status"`
	InstitutionName     *string            `json:"institution_name,omitempty"`
	MergedToID          *int               `json:"merged_to_id,omitempty"`
	DeletedAt           utils.NullableTime `json:"deleted_at,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	Profiles            []profile.Profile  `json:"profiles,omitempty"`
}
