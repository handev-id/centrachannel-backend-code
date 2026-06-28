package upload

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createTestFileHeader(t *testing.T, filename, content string) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, content); err != nil {
		t.Fatal(err)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(10 << 20); err != nil {
		t.Fatal(err)
	}
	return req.MultipartForm.File["file"][0]
}

func TestUploadService_Upload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storageResp := map[string]string{
			"public_url": "https://storage.example.com/uploads/test.txt",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "multipart/form-data") {
				t.Errorf("expected multipart/form-data, got %s", ct)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(storageResp)
		}))
		defer server.Close()

		svc := NewUploadService(server.URL)
		file := createTestFileHeader(t, "test.txt", "hello world")

		result, err := svc.Upload(file)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "test.txt" {
			t.Errorf("Name = %q, want %q", result.Name, "test.txt")
		}
		if result.Extname != "txt" {
			t.Errorf("Extname = %q, want %q", result.Extname, "txt")
		}
		if result.Size != 11 {
			t.Errorf("Size = %d, want %d", result.Size, 11)
		}
		if result.Type != "text/plain; charset=utf-8" {
			t.Errorf("Type = %q, want %q", result.Type, "text/plain; charset=utf-8")
		}
		if result.URL != "https://storage.example.com/uploads/test.txt" {
			t.Errorf("URL = %q, want %q", result.URL, "https://storage.example.com/uploads/test.txt")
		}
	})

	t.Run("storage returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "internal error")
		}))
		defer server.Close()

		svc := NewUploadService(server.URL)
		file := createTestFileHeader(t, "test.txt", "hello world")

		_, err := svc.Upload(file)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("storage returns malformed JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "not json")
		}))
		defer server.Close()

		svc := NewUploadService(server.URL)
		file := createTestFileHeader(t, "test.txt", "hello world")

		_, err := svc.Upload(file)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
