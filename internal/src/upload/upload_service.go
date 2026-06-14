package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type UploadResult struct {
	Message   string `json:"message"`
	Key       string `json:"key"`
	PublicURL string `json:"public_url"`
	Type      string `json:"type"`
	Size      int64  `json:"size"`
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

	var result UploadResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode storage response: %w", err)
	}

	return &result, nil
}
