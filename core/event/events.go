//go:build linux

package event

import "fmt"

type EventType int

const (
	EVENT_TYPE_UNKNOWN EventType = iota
	EVENT_TYPE_ACCEPT
	EVENT_TYPE_READ
	EVENT_TYPE_RECVMSG
	EVENT_TYPE_WRITE
	EVENT_TYPE_TIMEOUT
	EVENT_TYPE_CANCEL
	EVENT_TYPE_SENDMSG
	EVENT_TYPE_LAST
)

func (et EventType) String() string {
	switch et {
	case EVENT_TYPE_UNKNOWN:
		return "EVENT_TYPE_UNKNOWN"
	case EVENT_TYPE_ACCEPT:
		return "EVENT_TYPE_ACCEPT"
	case EVENT_TYPE_READ:
		return "EVENT_TYPE_READ"
	case EVENT_TYPE_WRITE:
		return "EVENT_TYPE_WRITE"
	case EVENT_TYPE_TIMEOUT:
		return "EVENT_TYPE_TIMEOUT"
	case EVENT_TYPE_RECVMSG:
		return "EVENT_TYPE_RECVMSG"
	case EVENT_TYPE_SENDMSG:
		return "EVENT_TYPE_SENDMSG"
	case EVENT_TYPE_CANCEL:
		return "EVENT_TYPE_CANCEL"
	case EVENT_TYPE_LAST:
		return "EVENT_TYPE_LAST"
	default:
		return fmt.Sprintf("UNKNOWN: %d", et)
	}
}
