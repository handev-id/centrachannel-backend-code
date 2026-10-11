package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"centrachannel/internal/src/message"
)

func TestHandleMessageUpdate_UnknownInstanceCannotMutate(t *testing.T) {
	called := false
	msgRepo := &mockMsgRepo{
		updateStatusByWebhookIDFunc: func(context.Context, message.DBTX, uuid.UUID, string, string) error {
			called = true
			return nil
		},
	}
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)
	data, err := json.Marshal(EvolutionMessageUpdate{KeyID: "shared-provider-id", Status: "DELIVERED"})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "unknown-instance", Data: data})
	if err == nil {
		t.Error("unknown instance must be rejected")
	}
	if called {
		t.Error("unknown instance reached message update")
	}
}
