package whatsapp

import "strings"

func ExtractPhoneFromJID(jid string) string {
	if idx := strings.Index(jid, "@"); idx != -1 {
		return jid[:idx]
	}
	return jid
}
