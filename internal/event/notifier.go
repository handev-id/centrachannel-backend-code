package event

type Notifier interface {
	Notify(tenantID int, event string, data interface{})
}
