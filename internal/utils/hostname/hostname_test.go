package hostname

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	for input, want := range map[string]string{" CRM.Company.COM. ": "crm.company.com", "demo.centrachannel.local": "demo.centrachannel.local", "xn--bcher-kva.example": "xn--bcher-kva.example"} {
		got, err := NormalizeName(input)
		if err != nil || got != want {
			t.Fatalf("%q = %q %v", input, got, err)
		}
	}
	for _, input := range []string{"", "https://crm.company.com", "crm.company.com:443", "crm.company.com/path", "crm.company.com?x=1", "*.company.com", "127.0.0.1", "[::1]", "company..com", "-company.com", "company-.com", "crm_company.com", "bücher.example", strings.Repeat("a", 64) + ".com", strings.Repeat("a.", 127) + "a"} {
		if _, err := NormalizeName(input); !errors.Is(err, ErrInvalidName) {
			t.Errorf("accepted %q", input)
		}
	}
	if _, err := ValidateName(" PLATFORM.TEST. ", "platform.test"); !errors.Is(err, ErrInvalidName) {
		t.Fatal("accepted reserved main domain")
	}
}
