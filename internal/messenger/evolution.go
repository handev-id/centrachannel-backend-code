package messenger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"centrachannel/internal/utils/media"

	"centrachannel/internal/utils/httplog"
	"centrachannel/internal/utils/logger"
)

type EvolutionConfig struct {
	APIURL   string
	APIKey   string
	DeviceID string
}

type EvolutionSender struct {
	cfg    EvolutionConfig
	client *http.Client
	logger *logger.Logger
}

func NewEvolutionSender(cfg EvolutionConfig, l *logger.Logger) *EvolutionSender {
	return &EvolutionSender{
		cfg: cfg,
		client: &http.Client{
			Transport: httplog.NewLoggingRoundTripper(http.DefaultTransport, l, "EvolutionSender"),
			Timeout:   60 * time.Second,
		},
		logger: l,
	}
}

func (s *EvolutionSender) Send(msg *OutgoingMessage) (string, error) {
	quoted, err := buildEvolutionQuote(msg.Quoted, msg.RecipientID+"@s.whatsapp.net")
	if err != nil {
		return "", err
	}
	if len(msg.Attachment) > 0 && string(msg.Attachment) != "null" {
		return s.sendMedia(msg, quoted)
	}
	if msg.Text != nil && strings.TrimSpace(*msg.Text) != "" {
		return s.sendText(msg, quoted)
	}
	return "", fmt.Errorf("no text or attachment provided")
}

func (s *EvolutionSender) sendText(msg *OutgoingMessage, quoted json.RawMessage) (string, error) {
	endpoint := fmt.Sprintf("%s/message/sendText/%s", strings.TrimRight(s.cfg.APIURL, "/"), s.cfg.DeviceID)

	payload := map[string]interface{}{
		"number": msg.RecipientID,
		"text":   *msg.Text,
		"delay":  1200,
	}
	if len(quoted) > 0 {
		payload["quoted"] = quoted
	}

	return s.doRequest(endpoint, payload)
}

func (s *EvolutionSender) sendMedia(msg *OutgoingMessage, quoted json.RawMessage) (string, error) {
	var att struct {
		URL      string `json:"url"`
		Type     string `json:"type"`
		Caption  string `json:"caption,omitempty"`
		FileName string `json:"fileName,omitempty"`
		Mimetype string `json:"mimetype,omitempty"`
	}
	if err := json.Unmarshal(msg.Attachment, &att); err != nil || !media.IsHTTPURL(att.URL) {
		return "", fmt.Errorf("invalid attachment data")
	}

	mediaType := att.Type
	if mediaType == "" {
		mediaType = "document"
	}
	if mediaType != "image" && mediaType != "video" && mediaType != "audio" && mediaType != "document" {
		return "", fmt.Errorf("unsupported attachment type: %s", mediaType)
	}
	if msg.Text != nil && strings.TrimSpace(*msg.Text) != "" {
		att.Caption = *msg.Text
	}

	endpoint := fmt.Sprintf("%s/message/sendMedia/%s", strings.TrimRight(s.cfg.APIURL, "/"), s.cfg.DeviceID)

	payload := map[string]interface{}{
		"number":    msg.RecipientID,
		"mediatype": mediaType,
		"media":     att.URL,
		"delay":     1200,
	}
	if len(quoted) > 0 {
		payload["quoted"] = quoted
	}
	if att.Caption != "" {
		payload["caption"] = att.Caption
	}
	if att.FileName != "" {
		payload["fileName"] = att.FileName
	}
	if att.Mimetype != "" {
		payload["mimetype"] = att.Mimetype
	}

	return s.doRequest(endpoint, payload)
}

type evolutionResponse struct {
	Status string `json:"status"`
	Key    struct {
		ID string `json:"id"`
	} `json:"key"`
	Error json.RawMessage `json:"error,omitempty"`
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", fmt.Errorf("read evolution response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("evolution returned HTTP %d", resp.StatusCode)
	}

	var evoResp evolutionResponse
	if err := json.Unmarshal(respBody, &evoResp); err != nil {
		return "", fmt.Errorf("failed to parse evolution api response: %w", err)
	}

	if len(evoResp.Error) > 0 && string(evoResp.Error) != "null" && string(evoResp.Error) != "false" && string(evoResp.Error) != `""` {
		return "", fmt.Errorf("evolution api error: %s", evoResp.Error)
	}

	if evoResp.Key.ID != "" {
		return evoResp.Key.ID, nil
	}

	return "", fmt.Errorf("evolution response has no message ID")
}
