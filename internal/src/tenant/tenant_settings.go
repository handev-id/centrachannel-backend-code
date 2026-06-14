package tenant

import "encoding/json"

type Settings struct {
	ChannelConfiguration *ChannelConfiguration `json:"channel_configuration,omitempty"`
}

type ChannelConfiguration struct {
	MetaAccessToken             string `json:"meta_access_token"`
	MetaPageID                  string `json:"meta_page_id,omitempty"`
	MetaInstagramBusinessID     string `json:"meta_instagram_business_id,omitempty"`
	WhatsappPhoneID             string `json:"whatsapp_phone_id"`
	EvolutionBusinessInstance  string `json:"evolution_business_instance,omitempty"`
}

func ParseSettings(raw json.RawMessage) (*Settings, error) {
	if len(raw) == 0 {
		return &Settings{}, nil
	}
	var s Settings
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
