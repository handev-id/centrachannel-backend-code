package messenger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"centrachannel/internal/utils/httplog"
	"centrachannel/internal/utils/logger"
)

const graphAPI = "https://graph.facebook.com/v22.0"

type MetaSender struct {
	cfg    MetaConfig
	client *http.Client
	logger *logger.Logger
}

func NewMetaSender(cfg MetaConfig, l *logger.Logger) *MetaSender {
	return &MetaSender{
		cfg: cfg,
		client: &http.Client{
			Transport: httplog.NewLoggingRoundTripper(http.DefaultTransport, l, "MetaSender"),
			Timeout:   30 * time.Second,
		},
		logger: l,
	}
}

func (s *MetaSender) SetAccessToken(token string) {
	s.cfg.AccessToken = token
}

func (s *MetaSender) SetWhatsappPhoneID(phoneID string) {
	s.cfg.WhatsappPhoneID = phoneID
}

func (s *MetaSender) Send(msg *OutgoingMessage) (string, error) {
	switch msg.ChannelType {
	case "facebook":
		return s.sendFacebook(msg)
	case "instagram":
		return s.sendInstagram(msg)
	case "whatsapp_business":
		return s.sendWhatsAppBusiness(msg)
	default:
		return "", fmt.Errorf("unsupported channel type for meta sender: %s", msg.ChannelType)
	}
}

func (s *MetaSender) sendFacebook(msg *OutgoingMessage) (string, error) {
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": msg.RecipientID},
		"message":   s.buildMessagePayload(msg),
	}
	return s.doRequest(fmt.Sprintf("%s/me/messages?access_token=%s", graphAPI, s.cfg.AccessToken), payload)
}

func (s *MetaSender) sendInstagram(msg *OutgoingMessage) (string, error) {
	// Instagram uses the same /me/messages endpoint as Facebook
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": msg.RecipientID},
		"message":   s.buildMessagePayload(msg),
	}
	return s.doRequest(fmt.Sprintf("%s/me/messages?access_token=%s", graphAPI, s.cfg.AccessToken), payload)
}

func (s *MetaSender) sendWhatsAppBusiness(msg *OutgoingMessage) (string, error) {
	if s.cfg.WhatsappPhoneID == "" {
		return "", fmt.Errorf("whatsapp phone number id not configured")
	}
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":               msg.RecipientID,
	}
	if msg.Text != nil {
		payload["type"] = "text"
		payload["text"] = map[string]string{"body": *msg.Text}
	} else if msg.Attachment != nil {
		payload["type"] = "attachment"
		var att struct {
			URL  string `json:"url"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg.Attachment, &att); err == nil && att.URL != "" {
			payload["attachment"] = map[string]interface{}{
				"type": att.Type,
				"payload": map[string]string{
					"url": att.URL,
				},
			}
		}
	}
	return s.doRequest(fmt.Sprintf("%s/%s/messages?access_token=%s", graphAPI, s.cfg.WhatsappPhoneID, s.cfg.AccessToken), payload)
}

func (s *MetaSender) buildMessagePayload(msg *OutgoingMessage) map[string]interface{} {
	if msg.Text != nil {
		return map[string]interface{}{"text": *msg.Text}
	}
	if msg.Attachment != nil {
		var att struct {
			URL  string `json:"url"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg.Attachment, &att); err == nil && att.URL != "" {
			return map[string]interface{}{
				"attachment": map[string]interface{}{
					"type": att.Type,
					"payload": map[string]string{
						"url": att.URL,
					},
				},
			}
		}
	}
	return map[string]interface{}{"text": ""}
}

type metaResponse struct {
	MessageID string `json:"message_id"`
	Message   struct {
		ID string `json:"id"`
	} `json:"message"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

func (s *MetaSender) doRequest(url string, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := s.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("meta api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var mr metaResponse
	if err := json.Unmarshal(respBody, &mr); err != nil {
		return "", fmt.Errorf("failed to parse meta api response: %w", err)
	}

	if mr.Error != nil {
		return "", fmt.Errorf("meta api error (code %d): %s", mr.Error.Code, mr.Error.Message)
	}

	// Facebook/Instagram return message_id, WhatsApp returns messages[].id
	if mr.MessageID != "" {
		return mr.MessageID, nil
	}
	if mr.Message.ID != "" {
		return mr.Message.ID, nil
	}
	if len(mr.Messages) > 0 {
		return mr.Messages[0].ID, nil
	}

	return "sent", nil
}
