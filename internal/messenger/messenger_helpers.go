package messenger

import (
	"encoding/json"
	"fmt"
)

func buildEvolutionQuote(quote *QuotedMessage, remoteJID string) (json.RawMessage, error) {
	if quote == nil {
		return nil, nil
	}
	content := map[string]interface{}{}
	if len(quote.Attachment) > 0 && string(quote.Attachment) != "null" {
		var att struct {
			Type       string `json:"type"`
			Mimetype   string `json:"mimetype"`
			Caption    string `json:"caption"`
			FileName   string `json:"fileName"`
			IsAnimated bool   `json:"is_animated"`
		}
		if err := json.Unmarshal(quote.Attachment, &att); err != nil {
			return nil, err
		}
		names := map[string]string{"image": "imageMessage", "video": "videoMessage", "audio": "audioMessage", "document": "documentMessage", "sticker": "stickerMessage"}
		name, ok := names[att.Type]
		if !ok {
			return nil, fmt.Errorf("cannot quote unsupported message type")
		}
		content[name] = map[string]interface{}{"mimetype": att.Mimetype, "caption": att.Caption, "fileName": att.FileName, "isAnimated": att.IsAnimated}
	} else if quote.Text != nil {
		content["conversation"] = *quote.Text
	} else {
		return nil, fmt.Errorf("reply target has no supported content")
	}
	return json.Marshal(map[string]interface{}{"key": map[string]interface{}{"id": quote.ProviderMessageID, "remoteJid": remoteJID, "fromMe": quote.FromMe}, "message": content})
}
