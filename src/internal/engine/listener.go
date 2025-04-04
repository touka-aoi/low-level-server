//go:build linux

package engine

import (
	"net/netip"

	"github.com/touka-aoi/low-level-server/internal/io"
)

type Listener interface {
	Fd() int32
	Close() error
}

type TCPListener struct {
	socket *io.Socket
}

func Listen(protocol, externalAddress string, listenMaxConnection int) (Listener, error) {
	switch protocol {
	case "tcp":
		addr, err := netip.ParseAddrPort(externalAddress)
		if err != nil {
			return nil, err
		}

		s := io.CreateSocket()
		s.Bind(addr)
		s.Listen(listenMaxConnection)

		return &TCPListener{
			socket: s,
		}, nil
	}

	return nil, nil
}

func (l *TCPListener) Close() error {
	return nil
}

func (l *TCPListener) Fd() int32 {
	return l.socket.Fd
}
