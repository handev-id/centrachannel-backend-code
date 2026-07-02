package whatsapp_device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"centrachannel/internal/utils/httplog"
	"centrachannel/internal/utils/logger"
)

type evolutionClient struct {
	apiURL string
	apiKey string
	logger *logger.Logger
	client *http.Client
}

func NewEvolutionClient(apiURL, apiKey string, logger *logger.Logger) WhatsAppClient {
	return &evolutionClient{
		apiURL: apiURL,
		apiKey: apiKey,
		logger: logger,
		client: &http.Client{
			Transport: httplog.NewLoggingRoundTripper(http.DefaultTransport, logger, "EvolutionClient"),
			Timeout:   30 * time.Second,
		},
	}
}

func (c *evolutionClient) SendMessage(ctx context.Context, device *WhatsAppDevice, to string, text string) (*MessageResult, error) {
	endpoint := fmt.Sprintf("%s/message/sendText/%s", strings.TrimRight(c.apiURL, "/"), device.WhatsappID)

	payload := map[string]interface{}{
		"number": to,
		"text":   text,
		"delay":  1200,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Status string `json:"status"`
		Key    struct {
			ID string `json:"id"`
		} `json:"key"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse evolution api response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("evolution api error: %s", result.Error)
	}

	msgResult := &MessageResult{
		MessageID: result.Key.ID,
		Status:    result.Status,
		Timestamp: time.Now(),
	}
	if msgResult.MessageID == "" {
		msgResult.MessageID = fmt.Sprintf("evo_%d", time.Now().UnixMilli())
	}

	return msgResult, nil
}

func (c *evolutionClient) GetQR(ctx context.Context, device *WhatsAppDevice) (string, error) {
	endpoint := fmt.Sprintf("%s/instance/connect/%s", strings.TrimRight(c.apiURL, "/"), device.WhatsappID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Base64 string `json:"base64"`
		Code   string `json:"code"`
		Error  string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse evolution api response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("evolution api error: %s", result.Error)
	}

	if result.Base64 != "" {
		return result.Base64, nil
	}
	if result.Code != "" {
		return result.Code, nil
	}

	return "", fmt.Errorf("no qr code returned from evolution api")
}

func (c *evolutionClient) CheckConnection(ctx context.Context, device *WhatsAppDevice) (bool, error) {
	endpoint := fmt.Sprintf("%s/instance/connectionState/%s", strings.TrimRight(c.apiURL, "/"), device.WhatsappID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Instance struct {
			State string `json:"state"`
		} `json:"instance"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("failed to parse evolution api response: %w", err)
	}

	if result.Error != "" {
		return false, fmt.Errorf("evolution api error: %s", result.Error)
	}

	return result.Instance.State == "open", nil
}

func (c *evolutionClient) CreateInstance(ctx context.Context, device *WhatsAppDevice) error {
	endpoint := fmt.Sprintf("%s/instance/create", strings.TrimRight(c.apiURL, "/"))

	payload := map[string]interface{}{
		"instanceName": device.WhatsappID,
		"integration":  "WHATSAPP-BAILEYS",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil
	}
	if result.Error != "" {
		return fmt.Errorf("evolution api error: %s", result.Error)
	}

	return nil
}

func (c *evolutionClient) DeleteInstance(ctx context.Context, device *WhatsAppDevice) error {
	endpoint := fmt.Sprintf("%s/instance/delete/%s", strings.TrimRight(c.apiURL, "/"), device.WhatsappID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil
	}
	if result.Error != "" {
		return fmt.Errorf("evolution api error: %s", result.Error)
	}

	return nil
}

func (c *evolutionClient) SetWebhook(ctx context.Context, device *WhatsAppDevice, webhookURL string) error {
	endpoint := fmt.Sprintf("%s/instance/setWebhook/%s", strings.TrimRight(c.apiURL, "/"), device.WhatsappID)

	payload := map[string]interface{}{
		"url":     webhookURL,
		"enabled": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil
	}
	if result.Error != "" {
		return fmt.Errorf("evolution api error: %s", result.Error)
	}

	return nil
}

func (c *evolutionClient) Disconnect(ctx context.Context, device *WhatsAppDevice) error {
	endpoint := fmt.Sprintf("%s/instance/delete/%s", strings.TrimRight(c.apiURL, "/"), device.WhatsappID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("evolution api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse evolution api response: %w", err)
	}

	if result.Error != "" {
		return fmt.Errorf("evolution api error: %s", result.Error)
	}

	return nil
}
