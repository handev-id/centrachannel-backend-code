package hostname

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var ErrInvalidName = errors.New("Invalid domain name")

// NormalizeName validates and normalizes a complete ASCII hostname.
func NormalizeName(name string) (string, error) {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	if name == "" || len(name) > 253 || net.ParseIP(name) != nil {
		return "", ErrInvalidName
	}
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrInvalidName
		}
		for _, ch := range label {
			if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
				return "", ErrInvalidName
			}
		}
	}
	return name, nil
}
func ValidateName(name, baseDomain string) (string, error) {
	name, err := NormalizeName(name)
	if err != nil {
		return "", err
	}
	base, _ := NormalizeName(baseDomain)
	if name == base {
		return "", fmt.Errorf("%w: application base domain is reserved", ErrInvalidName)
	}
	return name, nil
}
