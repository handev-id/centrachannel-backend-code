package messenger

import (
	"fmt"

	"centrachannel/internal/utils/logger"
)

func NewSender(channelType string, cfg MetaConfig, l *logger.Logger, evoCfg ...EvolutionConfig) (Messenger, error) {
	switch channelType {
	case "facebook", "instagram":
		return NewMetaSender(cfg, l), nil
	case "whatsapp_business":
		if len(evoCfg) > 0 && evoCfg[0].APIURL != "" && evoCfg[0].DeviceID != "" {
			return NewEvolutionSender(evoCfg[0], l), nil
		}
		return NewMetaSender(cfg, l), nil
	case "whatsapp":
		if len(evoCfg) > 0 && evoCfg[0].APIURL != "" {
			return NewEvolutionSender(evoCfg[0], l), nil
		}
		return NewMockSender(), nil
	default:
		return nil, fmt.Errorf("unknown channel type: %s", channelType)
	}
}

type MetaConfig struct {
	AccessToken     string `json:"meta_access_token"`
	WhatsappPhoneID string `json:"whatsapp_phone_id"`
}
