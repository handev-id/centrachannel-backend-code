package messenger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type EvolutionConfig struct {
	APIURL   string
	APIKey   string
	DeviceID string
}

type EvolutionSender struct {
	cfg    EvolutionConfig
	client *http.Client
}

func NewEvolutionSender(cfg EvolutionConfig) *EvolutionSender {
	return &EvolutionSender{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *EvolutionSender) Send(msg *OutgoingMessage) (string, error) {
	if msg.Text != nil {
		return s.sendText(msg)
	}
	if msg.Attachment != nil {
		return s.sendMedia(msg)
	}
	return "", fmt.Errorf("no text or attachment provided")
}

func (s *EvolutionSender) sendText(msg *OutgoingMessage) (string, error) {
	endpoint := fmt.Sprintf("%s/message/sendText/%s", strings.TrimRight(s.cfg.APIURL, "/"), s.cfg.DeviceID)

	payload := map[string]interface{}{
		"number": msg.RecipientID,
		"text":   *msg.Text,
		"delay":  1200,
	}

	return s.doRequest(endpoint, payload)
}

func (s *EvolutionSender) sendMedia(msg *OutgoingMessage) (string, error) {
	var att struct {
		URL      string `json:"url"`
		Type     string `json:"type"`
		Caption  string `json:"caption,omitempty"`
		FileName string `json:"fileName,omitempty"`
	}
	if err := json.Unmarshal(msg.Attachment, &att); err != nil || att.URL == "" {
		return "", fmt.Errorf("invalid attachment data")
	}

	mediaType := att.Type
	if mediaType == "" {
		mediaType = "document"
	}

	endpoint := fmt.Sprintf("%s/message/sendMedia/%s", strings.TrimRight(s.cfg.APIURL, "/"), s.cfg.DeviceID)

	payload := map[string]interface{}{
		"number":    msg.RecipientID,
		"mediatype": mediaType,
		"media":     att.URL,
		"delay":     1200,
	}
	if att.Caption != "" {
		payload["caption"] = att.Caption
	}
	if att.FileName != "" {
		payload["fileName"] = att.FileName
	}

	return s.doRequest(endpoint, payload)
}

type evolutionResponse struct {
	Status  string `json:"status"`
	Key     struct {
		ID string `json:"id"`
	} `json:"key"`
	Error string `json:"error,omitempty"`
}

func (s *EvolutionSender) doRequest(url string, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var evoResp evolutionResponse
	if err := json.Unmarshal(respBody, &evoResp); err != nil {
		return "", fmt.Errorf("failed to parse evolution api response: %w", err)
	}

	if evoResp.Error != "" {
		return "", fmt.Errorf("evolution api error: %s", evoResp.Error)
	}

	if evoResp.Key.ID != "" {
		return evoResp.Key.ID, nil
	}

	return "sent", nil
}
