package ws

import "github.com/google/uuid"

type Notifier interface {
	Notify(tenantID uuid.UUID, event string, data interface{})
}
