package ws

type MockNotifier struct{}

func NewMockNotifier() Notifier {
	return &MockNotifier{}
}

func (m *MockNotifier) Notify(tenantID int, event string, data interface{}) {}
