package apis

type EventType string

const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
	Synced   EventType = "SYNCED" //Added or Modified
)

type Event struct {
	Type   EventType
	Object interface{}
}
