package whatsapp

import "testing"

func TestExtractPhoneFromJID(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"5511999999999@s.whatsapp.net", "5511999999999"},
		{"12036301234567@g.us", "12036301234567"},
		{"5511999999999", "5511999999999"},
		{"", ""},
	}
	for _, tt := range tests {
		result := ExtractPhoneFromJID(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractPhoneFromJID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
