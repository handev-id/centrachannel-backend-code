package whatsapp_device

type CreateDeviceRequest struct {
	Name        string `json:"name" validate:"required"`
	CountryCode string `json:"country_code" validate:"required"`
	Phone       string `json:"phone" validate:"required"`
	WhatsappID  string `json:"whatsapp_id" validate:"required"`
}

type UpdateDeviceRequest struct {
	Name        *string `json:"name,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Status      *string `json:"status,omitempty"`
}
