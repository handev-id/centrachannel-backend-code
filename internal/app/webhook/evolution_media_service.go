package webhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"centrachannel/internal/src/message"
	"centrachannel/internal/utils/media"
	"centrachannel/internal/utils/storage"
)

type evolutionMediaService struct {
	apiURL  string
	apiKey  string
	http    *http.Client
	storage *storage.Client
}

func (s *evolutionMediaService) Store(ctx context.Context, instance string, data *EvolutionMessageUpsert, attachment json.RawMessage) (json.RawMessage, error) {
	var source struct {
		message.Attachment
		Base64 string `json:"base64"`
	}
	if err := json.Unmarshal(attachment, &source); err != nil {
		return nil, err
	}
	if source.Type == "unsupported" {
		return attachment, nil
	}
	att := source.Attachment
	att.URL = ""
	att.Status = "failed"
	if source.Base64 == "" {
		var original interface{} = data.Message
		if len(data.RawMessage) > 0 {
			original = data.RawMessage
		}
		body, err := json.Marshal(map[string]interface{}{"message": map[string]interface{}{"key": data.Key, "message": original}, "convertToMp4": false})
		if err != nil {
			return nil, err
		}
		endpoint := strings.TrimRight(s.apiURL, "/") + "/chat/getBase64FromMediaMessage/" + url.PathEscape(instance)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", s.apiKey)
		resp, err := s.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download Evolution media: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("download Evolution media: HTTP %d", resp.StatusCode)
		}
		var result struct {
			Base64   string `json:"base64"`
			Mimetype string `json:"mimetype"`
			FileName string `json:"fileName"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, media.MaxRequestSize)).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode Evolution media: %w", err)
		}
		source.Base64 = result.Base64
		if result.Mimetype != "" {
			att.Mimetype = result.Mimetype
		}
		if result.FileName != "" {
			att.FileName = result.FileName
		}
	}
	if source.Base64 == "" {
		return nil, fmt.Errorf("Evolution returned no media content")
	}
	if strings.HasPrefix(source.Base64, "data:") {
		_, encoded, ok := strings.Cut(source.Base64, ",")
		if !ok {
			return nil, fmt.Errorf("invalid media data URL")
		}
		source.Base64 = encoded
	}
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(source.Base64))
	sample := make([]byte, 512)
	n, err := io.ReadFull(decoder, sample)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("decode media base64: %w", err)
	}
	if att.FileName == "" {
		ext := ""
		if extensions, err := mime.ExtensionsByType(att.Mimetype); err == nil && len(extensions) > 0 {
			ext = extensions[0]
		}
		att.FileName = data.Key.ID + ext
	}
	att.FileName = filepath.Base(att.FileName)
	if att.Mimetype == "" {
		att.Mimetype = media.MIMEType(att.FileName, sample[:n])
	}
	storedURL, size, err := s.storage.Upload(ctx, att.FileName, io.MultiReader(bytes.NewReader(sample[:n]), decoder))
	if errors.Is(err, media.ErrTooLarge) {
		att.Status = "too_large"
		return json.Marshal(att)
	}
	if err != nil {
		return nil, err
	}
	att.URL, att.Size, att.Status = storedURL, size, "ready"
	return json.Marshal(att)
}

func newEvolutionMediaService(apiURL, apiKey string, client *storage.Client) *evolutionMediaService {
	return &evolutionMediaService{apiURL: apiURL, apiKey: apiKey, http: &http.Client{Timeout: 60 * time.Second}, storage: client}
}
