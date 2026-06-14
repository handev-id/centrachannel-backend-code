package profile

type ListProfileQuery struct {
	ContactID int `query:"contact_id"`
	ChannelID int `query:"channel_id"`
}

type UpdateProfileRequest struct {
	IsMain                 *bool   `json:"is_main,omitempty"`
	LinkedDeviceWhatsappID *string `json:"linked_device_whatsapp_id,omitempty"`
	DisplayName            *string `json:"display_name,omitempty"`
}
