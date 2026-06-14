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
		expected := UploadResult{
			Message:   "File uploaded",
			Key:       "uploads/test.txt",
			PublicURL: "https://storage.example.com/uploads/test.txt",
			Type:      "text/plain",
			Size:      11,
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
			json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		svc := NewUploadService(server.URL)
		file := createTestFileHeader(t, "test.txt", "hello world")

		result, err := svc.Upload(file)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Message != expected.Message {
			t.Errorf("Message = %q, want %q", result.Message, expected.Message)
		}
		if result.Key != expected.Key {
			t.Errorf("Key = %q, want %q", result.Key, expected.Key)
		}
		if result.PublicURL != expected.PublicURL {
			t.Errorf("PublicURL = %q, want %q", result.PublicURL, expected.PublicURL)
		}
		if result.Type != expected.Type {
			t.Errorf("Type = %q, want %q", result.Type, expected.Type)
		}
		if result.Size != expected.Size {
			t.Errorf("Size = %d, want %d", result.Size, expected.Size)
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
