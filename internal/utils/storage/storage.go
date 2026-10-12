package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"centrachannel/internal/utils/logger"
	"centrachannel/internal/utils/media"
)

type Client struct {
	url    string
	secret string
	http   *http.Client
	logger *logger.Logger
}

func NewClient(url, secret string, log *logger.Logger) *Client {
	return &Client{url: url, secret: secret, http: &http.Client{Timeout: 60 * time.Second}, logger: log}
}

// Upload spools multipart to disk instead of copying large files into RAM.
func (c *Client) Upload(ctx context.Context, name string, src io.Reader) (string, int64, error) {
	body, err := os.CreateTemp("", "centrachannel-upload-*")
	if err != nil {
		return "", 0, fmt.Errorf("create upload buffer: %w", err)
	}
	defer os.Remove(body.Name())
	defer body.Close()
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		return "", 0, err
	}
	size, err := io.Copy(part, io.LimitReader(src, media.MaxFileSize+1))
	if err != nil {
		return "", 0, fmt.Errorf("read upload: %w", err)
	}
	if size > media.MaxFileSize {
		return "", 0, media.ErrTooLarge
	}
	if size == 0 {
		return "", 0, fmt.Errorf("file is empty")
	}
	if err := w.Close(); err != nil {
		return "", 0, err
	}
	length, err := body.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", 0, err
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, body)
	if err != nil {
		return "", 0, err
	}
	req.ContentLength = length
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.secret)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("upload to storage: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", 0, fmt.Errorf("storage returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		URL string `json:"public_url"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&result); err != nil {
		return "", 0, fmt.Errorf("decode storage response: %w", err)
	}
	if !media.IsHTTPURL(result.URL) {
		return "", 0, fmt.Errorf("storage did not return a valid public URL")
	}
	if c.logger != nil {
		c.logger.Info("stored media file: size=%d", size)
	}
	return result.URL, size, nil
}
