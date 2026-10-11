package ws

import "github.com/google/uuid"

type MockNotifier struct{}

func NewMockNotifier() Notifier {
	return &MockNotifier{}
}

func (m *MockNotifier) Notify(tenantID uuid.UUID, event string, data interface{}) {}
