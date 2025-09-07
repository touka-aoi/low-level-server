package peer

// 参考: https://go.googlesource.com/go/%2B/master/src/net/http/server.go#3267
type ConnState int32

const (
	StateNew    ConnState = iota
	StateActive           // has data transfer
	StateIdle             // keep-alive
	StateClosed
)

var stateName = map[ConnState]string{
	StateNew:    "new",
	StateActive: "active",
	StateIdle:   "idle",
	StateClosed: "closed",
}

func (s ConnState) String() string {
	return stateName[s]
}
