package internal

type EventType int

const (
	EVENT_TYPE_ACCEPT EventType = iota
	EVENT_TYPE_READ
	EVENT_TYPE_WRITE
)

func (et EventType) String() string {
	switch et {
	case EVENT_TYPE_ACCEPT:
		return "EVENT_TYPE_ACCEPT"
	case EVENT_TYPE_READ:
		return "EVENT_TYPE_READ"
	case EVENT_TYPE_WRITE:
		return "EVENT_TYPE_WRITE"
	default:
		return "UNKNOWN"
	}
}
