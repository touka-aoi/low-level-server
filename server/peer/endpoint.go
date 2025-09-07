package peer

import "net/netip"

type Endpoint interface {
	Fd() int32
	LocalAddr() netip.AddrPort
	RemoteAddr() netip.AddrPort
	Status() string
}
