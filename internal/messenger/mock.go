package messenger

import (
	"fmt"
	"time"
)

type MockSender struct{}

func NewMockSender() *MockSender {
	return &MockSender{}
}

func (s *MockSender) Send(msg *OutgoingMessage) (string, error) {
	return fmt.Sprintf("mock_%d", time.Now().UnixMilli()), nil
}
