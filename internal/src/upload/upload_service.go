package upload

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"centrachannel/internal/utils/logger"
	"centrachannel/internal/utils/media"
	"centrachannel/internal/utils/storage"
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
	storage *storage.Client
}

func NewUploadService(storageURL, storageSecretKey string, l *logger.Logger) UploadService {
	return &uploadService{storage: storage.NewClient(storageURL, storageSecretKey, l)}
}

func (s *uploadService) Upload(file *multipart.FileHeader) (*UploadResult, error) {
	if file.Size > media.MaxFileSize {
		return nil, media.ErrTooLarge
	}
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer src.Close()
	sample := make([]byte, 512)
	n, err := src.Read(sample)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	name := filepath.Base(file.Filename)
	url, size, err := s.storage.Upload(context.Background(), name, src)
	if err != nil {
		return nil, err
	}
	return &UploadResult{
		Name: name, Extname: strings.TrimPrefix(filepath.Ext(name), "."),
		Size: size, Type: media.MIMEType(name, sample[:n]), URL: url,
	}, nil
}
