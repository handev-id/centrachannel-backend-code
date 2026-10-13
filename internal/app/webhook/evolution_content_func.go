package webhook

import (
	"centrachannel/internal/utils/whatsapp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"

	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/message"
	"errors"
)

func extractEvolutionQuote(raw json.RawMessage, contextInfo *EvolutionContextInfo, key EvolutionMessageKey) *message.Quote {
	if contextInfo == nil {
		var content map[string]json.RawMessage
		if json.Unmarshal(raw, &content) != nil {
			return nil
		}
		for _, name := range []string{"ephemeralMessage", "documentWithCaptionMessage", "viewOnceMessage", "viewOnceMessageV2", "viewOnceMessageV2Extension"} {
			if wrapper, ok := content[name]; ok {
				var wrapped struct {
					Message json.RawMessage `json:"message"`
				}
				if json.Unmarshal(wrapper, &wrapped) == nil {
					return extractEvolutionQuote(wrapped.Message, nil, key)
				}
			}
		}
		for _, value := range content {
			var item struct {
				ContextInfo *EvolutionContextInfo `json:"contextInfo"`
			}
			if json.Unmarshal(value, &item) == nil && item.ContextInfo != nil && item.ContextInfo.StanzaID != "" {
				contextInfo = item.ContextInfo
				break
			}
		}
	}
	if contextInfo == nil || contextInfo.StanzaID == "" {
		return nil
	}
	quote := &message.Quote{ProviderMessageID: contextInfo.StanzaID}
	if contextInfo.Participant != "" {
		if contextInfo.Participant == key.RemoteJid || contextInfo.Participant == key.RemoteJidAlt {
			quote.SenderType = "contact"
		} else if strings.HasSuffix(contextInfo.Participant, "@s.whatsapp.net") && strings.HasSuffix(key.RemoteJid, "@s.whatsapp.net") {
			quote.SenderType = "user"
		}
	}
	var quoted EvolutionMessage
	if json.Unmarshal(contextInfo.QuotedMessage, &quoted) != nil {
		return quote
	}
	var contents map[string]json.RawMessage
	_ = json.Unmarshal(contextInfo.QuotedMessage, &contents)
	kind := ""
	for name := range contents {
		if name != "messageContextInfo" {
			kind = name
			break
		}
	}
	quote.Text, quote.Attachment = extractMessageContent(quoted, kind)
	if len(quote.Attachment) > 0 {
		var att map[string]interface{}
		if json.Unmarshal(quote.Attachment, &att) == nil {
			// Quoted provider URLs are encrypted; render a type/filename snapshot.
			delete(att, "url")
			delete(att, "base64")
			quote.Attachment, _ = json.Marshal(att)
		}
	}
	return quote
}

func isEvolutionGroup(jid string) bool {
	return strings.HasSuffix(jid, "@g.us") || jid == "status@broadcast"
}

func (value *EvolutionTimestamp) UnmarshalJSON(raw []byte) error {
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		parsed, err := number.Int64()
		if err != nil {
			return err
		}
		*value = EvolutionTimestamp(parsed)
		return nil
	}
	var parts struct {
		Low  uint32 `json:"low"`
		High int32  `json:"high"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return err
	}
	*value = EvolutionTimestamp(int64(parts.High)<<32 | int64(parts.Low))
	return nil
}

func evolutionEventTime(value EvolutionTimestamp) time.Time {
	if value == 0 {
		return time.Now().UTC()
	}
	if value > 100_000_000_000 {
		return time.UnixMilli(int64(value)).UTC()
	}
	return time.Unix(int64(value), 0).UTC()
}

func unwrapEvolutionMessage(content EvolutionMessage) EvolutionMessage {
	for {
		var inner *EvolutionMessage
		switch {
		case content.EphemeralMessage != nil:
			inner = content.EphemeralMessage.Message
		case content.DocumentWithCaptionMessage != nil:
			inner = content.DocumentWithCaptionMessage.Message
		case content.ViewOnceMessage != nil:
			inner = content.ViewOnceMessage.Message
		case content.ViewOnceMessageV2 != nil:
			inner = content.ViewOnceMessageV2.Message
		case content.ViewOnceMessageV2Extension != nil:
			inner = content.ViewOnceMessageV2Extension.Message
		}
		if inner == nil {
			return content
		}
		previous := content.Base64
		content = *inner
		if previous != "" {
			content.Base64 = previous
		}
	}
}

func (s *webhookService) applyEvolutionContentMutation(ctx context.Context, tenantID, profileID, channelID uuid.UUID, key EvolutionMessageKey, content EvolutionMessage, timestamp EvolutionTimestamp) (bool, error) {
	content = unwrapEvolutionMessage(content)
	event := message.Mutation{EventID: key.ID, EventAt: evolutionEventTime(timestamp), ActorKey: "contact", SenderType: "contact"}
	if key.FromMe {
		event.ActorKey = "user"
		event.SenderType = "user"
	}
	switch {
	case content.ReactionMessage != nil:
		reaction := content.ReactionMessage
		event.Kind = "reaction"
		event.TargetProviderID = reaction.Key.ID
		event.Emoji = reaction.Text
		if reaction.SenderTimestampMS != 0 {
			event.EventAt = evolutionEventTime(reaction.SenderTimestampMS)
		}
	case content.EditedMessage != nil && content.EditedMessage.Message != nil:
		event.Kind = "edit"
		event.TargetProviderID = key.ID
		edited := unwrapEvolutionMessage(*content.EditedMessage.Message)
		event.Text, _ = extractMessageContent(edited, "")
		if edited.Conversation == nil && edited.ExtendedTextMessage == nil && edited.ImageMessage == nil && edited.VideoMessage == nil && edited.DocumentMessage == nil {
			return true, fmt.Errorf("unsupported edited content")
		}
		encoded, _ := json.Marshal(content.EditedMessage)
		digest := sha256.Sum256(encoded)
		event.EventID = key.ID + ":" + event.EventAt.Format(time.RFC3339Nano) + ":" + hex.EncodeToString(digest[:])
	case content.ProtocolMessage != nil && content.ProtocolMessage.EditedMessage != nil:
		protocol := content.ProtocolMessage
		inner := EvolutionMessage{EditedMessage: &EvolutionMessageContainer{Message: protocol.EditedMessage}}
		if protocol.Key.ID != "" {
			key.ID = protocol.Key.ID
		}
		if protocol.TimestampMS != 0 {
			timestamp = protocol.TimestampMS
		}
		return s.applyEvolutionContentMutation(ctx, tenantID, profileID, channelID, key, inner, timestamp)
	case content.SecretEncryptedMessage != nil:
		return true, fmt.Errorf("encrypted edit requires the Evolution secret-edit adapter")
	default:
		return false, nil
	}
	if event.TargetProviderID == "" {
		return true, fmt.Errorf("message mutation has no target")
	}
	updated, err := s.msgRepo.ApplyMutation(ctx, s.db, tenantID, profileID, channelID, event)
	if err != nil {
		return true, err
	}
	if updated != nil && s.notifier != nil {
		s.notifier.Notify(tenantID, "message-updated", updated)
		if event.Kind == "edit" {
			s.notifier.Notify(tenantID, "conversation-updated", map[string]interface{}{"id": updated.ConversationID})
		}
	}
	return true, nil
}

func (s *webhookService) handleEvolutionEditedUpdate(ctx context.Context, payload *EvolutionWebhookPayload, data EvolutionMessageUpdate) error {
	if isEvolutionGroup(data.RemoteJid) {
		return nil
	}
	device, err := s.deviceRepo.GetByWhatsappID(ctx, s.db, payload.Instance)
	if err != nil {
		return err
	}
	if device == nil {
		return fmt.Errorf("device not found")
	}
	phone := whatsapp.ExtractPhoneFromJID(data.RemoteJid)
	if phone == "" {
		return fmt.Errorf("edited message has no peer")
	}
	channel, err := s.channelRepo.GetByType(ctx, s.db, "whatsapp")
	if err != nil {
		return err
	}
	profile, err := s.profileRepo.GetByExternalIDAndChannelID(ctx, s.db, device.TenantID, phone, channel.ID)
	if err != nil {
		return err
	}
	if profile == nil {
		return fmt.Errorf("edited message profile not found")
	}
	timestamp := data.MessageTimestamp
	if data.TimestampMs != 0 {
		timestamp = data.TimestampMs
	}
	_, err = s.applyEvolutionContentMutation(ctx, device.TenantID, profile.ID, channel.ID, EvolutionMessageKey{ID: data.KeyID, RemoteJid: data.RemoteJid, FromMe: data.FromMe}, *data.Message, timestamp)
	return err
}

func (s *webhookService) populateEvolutionQuote(ctx context.Context, msg *message.Message, data EvolutionMessageUpsert, profileID, channelID uuid.UUID, peer *contact.Contact) error {
	quote := extractEvolutionQuote(data.RawMessage, data.ContextInfo, data.Key)
	if quote == nil {
		return nil
	}
	target, err := s.msgRepo.GetByProviderID(ctx, s.db, msg.TenantID, profileID, channelID, quote.ProviderMessageID)
	if err != nil && !errors.Is(err, message.ErrMessageNotFound) {
		return fmt.Errorf("resolve quote: %w", err)
	}
	if target != nil {
		quote.MessageID = &target.ID
		quote.SenderType = target.SenderType
		var stored, snapshot message.Attachment
		if json.Unmarshal(target.Attachment, &stored) == nil && json.Unmarshal(quote.Attachment, &snapshot) == nil && stored.Type == snapshot.Type {
			snapshot.URL = stored.URL
			snapshot.Status = stored.Status
			snapshot.Size = stored.Size
			quote.Attachment, err = json.Marshal(snapshot)
			if err != nil {
				return err
			}
		}
	}
	if quote.SenderType == "contact" {
		quote.SenderName = peer.FirstName
		if peer.LastName != nil {
			quote.SenderName += " " + *peer.LastName
		}
	}
	msg.Reply = quote
	msg.WebhookMessageReplyID = &quote.ProviderMessageID
	return nil
}
