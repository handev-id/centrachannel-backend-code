package avatar

import (
	"encoding/json"
	"fmt"
)

const diceBearURL = "https://api.dicebear.com/9.x/initials/svg"

type Avatar struct {
	URL string `json:"url"`
}

func GenerateInitials(name string) json.RawMessage {
	avatar := Avatar{
		URL: fmt.Sprintf("%s?seed=%s", diceBearURL, name),
	}
	data, _ := json.Marshal(avatar)
	return data
}
