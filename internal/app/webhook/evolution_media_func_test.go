package webhook

import (
	"encoding/json"
	"testing"
)

func TestExtractMessageContent(t *testing.T) {
	// conversation
	text := "Hello"
	txt, att := extractMessageContent(EvolutionMessage{Conversation: &text}, "conversation")
	if txt == nil || *txt != "Hello" { t.Errorf("expected 'Hello', got %v", txt) }
	if att != nil { t.Error("expected nil attachment") }

	// extendedTextMessage
	msg := EvolutionMessage{ExtendedTextMessage: &struct{ Text string `json:"text"` }{Text: "Extended"}}
	txt, att = extractMessageContent(msg, "extendedTextMessage")
	if txt == nil || *txt != "Extended" { t.Errorf("expected 'Extended', got %v", txt) }
	if att != nil { t.Error("expected nil attachment") }

	// image with caption
	imgMsg := EvolutionMessage{ImageMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Caption string `json:"caption,omitempty"`; Base64 string `json:"base64,omitempty"`
	}{URL: "https://img.url/p.jpg", Mimetype: "image/jpeg", Caption: "Nice pic"}}
	txt, att = extractMessageContent(imgMsg, "imageMessage")
	if txt == nil || *txt != "Nice pic" { t.Errorf("expected 'Nice pic', got %v", txt) }
	if att == nil { t.Fatal("expected attachment") }

	// image without caption
	imgNoCap := EvolutionMessage{ImageMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Caption string `json:"caption,omitempty"`; Base64 string `json:"base64,omitempty"`
	}{URL: "https://img.url/p.jpg", Mimetype: "image/jpeg"}}
	txt, att = extractMessageContent(imgNoCap, "imageMessage")
	if txt != nil { t.Errorf("expected nil text, got %v", *txt) }
	if att == nil { t.Fatal("expected attachment") }

	// video
	vidMsg := EvolutionMessage{VideoMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Caption string `json:"caption,omitempty"`; Base64 string `json:"base64,omitempty"`
	}{URL: "https://vid.url/v.mp4", Mimetype: "video/mp4", Caption: "Check this"}}
	txt, att = extractMessageContent(vidMsg, "videoMessage")
	if txt == nil || *txt != "Check this" { t.Errorf("expected 'Check this', got %v", txt) }
	if att == nil { t.Fatal("expected attachment") }

	// audio
	audMsg := EvolutionMessage{AudioMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Base64 string `json:"base64,omitempty"`
	}{URL: "https://audio.url/v.ogg", Mimetype: "audio/ogg"}}
	txt, att = extractMessageContent(audMsg, "audioMessage")
	if txt != nil { t.Errorf("expected nil text, got %v", *txt) }
	if att == nil { t.Fatal("expected attachment") }

	// document
	docMsg := EvolutionMessage{DocumentMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; FileName string `json:"fileName,omitempty"`; Base64 string `json:"base64,omitempty"`
	}{URL: "https://doc.url/r.pdf", Mimetype: "application/pdf", FileName: "report.pdf"}}
	txt, att = extractMessageContent(docMsg, "documentMessage")
	if txt != nil { t.Errorf("expected nil text, got %v", *txt) }
	if att == nil { t.Fatal("expected attachment") }

	// empty
	txt, att = extractMessageContent(EvolutionMessage{}, "")
	if txt != nil || att != nil { t.Error("expected nil text and nil attachment") }
}

func TestExtractMessageContent_Base64Fallback(t *testing.T) {
	// image with base64 but no valid URL (CDN path)
	imgMsg := EvolutionMessage{ImageMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Caption string `json:"caption,omitempty"`; Base64 string `json:"base64,omitempty"`
	}{URL: "/v/t62.11036-24/1198/HASH/file.jpg", Mimetype: "image/jpeg", Base64: "AAAAHGZ0eXBpc29"}}
	txt, att := extractMessageContent(imgMsg, "imageMessage")
	if txt != nil { t.Errorf("expected nil text, got %v", *txt) }
	if att == nil { t.Fatal("expected attachment") }

	// verify base64 is included
	var attMap map[string]interface{}
	if err := json.Unmarshal(att, &attMap); err != nil {
		t.Fatalf("failed to unmarshal attachment: %v", err)
	}
	if attMap["base64"] != "AAAAHGZ0eXBpc29" {
		t.Errorf("expected base64 data, got %v", attMap["base64"])
	}
	url, ok := attMap["url"].(string)
	if !ok || url == "" {
		t.Error("expected url to be set from base64 data URL")
	}
}
