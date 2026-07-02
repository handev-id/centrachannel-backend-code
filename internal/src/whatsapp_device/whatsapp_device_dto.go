package whatsapp_device

type CreateDeviceRequest struct {
	Name        string `json:"name" validate:"required,max=255"`
	CountryCode string `json:"country_code" validate:"required,max=10"`
	Phone       string `json:"phone" validate:"required,max=20"`
}

type UpdateDeviceRequest struct {
	Name        *string `json:"name,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Status      *string `json:"status,omitempty"`
}
