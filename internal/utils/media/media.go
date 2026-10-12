package media

import (
	"errors"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// Decimal bytes match the MB limit displayed in the chat composer.
const MaxFileSize int64 = 20_000_000

// Allows a base64-encoded 20 MB file plus webhook metadata.
const MaxRequestSize = 28_000_000

var ErrTooLarge = errors.New("file exceeds the 20 MB limit")

func IsHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}

func MIMEType(name string, sample []byte) string {
	detected := http.DetectContentType(sample)
	declared := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	// Extensions distinguish containers such as m4a/mp4 and docx/zip.
	if declared != "" && (detected == "application/octet-stream" || detected == "application/zip" || detected == "video/mp4" || (strings.HasPrefix(detected, "text/plain") && strings.HasPrefix(declared, "text/"))) {
		return declared
	}
	return detected
}

func Type(mimetype string) string {
	for _, kind := range []string{"image", "video", "audio"} {
		if strings.HasPrefix(mimetype, kind+"/") {
			return kind
		}
	}
	return "document"
}
