package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
)

type UploadResult struct {
	Name    string `json:"name"`
	Extname string `json:"extname"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	URL     string `json:"url"`
}

type UploadService interface {
	Upload(file *multipart.FileHeader) (*UploadResult, error)
}

type uploadService struct {
	storageURL string
}

func NewUploadService(storageURL string) UploadService {
	return &uploadService{storageURL: storageURL}
}

func (s *uploadService) Upload(file *multipart.FileHeader) (*UploadResult, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", file.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, src); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}
	writer.Close()

	resp, err := http.Post(s.storageURL, writer.FormDataContentType(), &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to storage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("storage returned %d: %s", resp.StatusCode, string(body))
	}

	var storageResp struct {
		PublicURL string `json:"public_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&storageResp); err != nil {
		return nil, fmt.Errorf("failed to decode storage response: %w", err)
	}

	name := file.Filename
	var extname string
	if idx := strings.LastIndex(name, "."); idx != -1 {
		extname = name[idx+1:]
	}
	mediaType := mime.TypeByExtension("." + extname)
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	return &UploadResult{
		Name:    name,
		Extname: extname,
		Size:    file.Size,
		Type:    mediaType,
		URL:     storageResp.PublicURL,
	}, nil
}
