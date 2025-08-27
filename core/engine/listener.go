//go:build linux

package engine

import (
	"net/netip"

	"github.com/touka-aoi/low-level-server/core/core"
)

type Listener interface {
	Fd() int32
	Close() error
}

type TCPListener struct {
	socket *core.Socket
}

type UDPListener struct {
	socket *core.Socket
}

func Listen(protocol, externalAddress string, listenMaxConnection int) (Listener, error) {
	switch protocol {
	case "tcp":
		addr, err := netip.ParseAddrPort(externalAddress)
		if err != nil {
			return nil, err
		}

		s := core.CreateTCPSocket()
		s.Bind(addr)
		err = s.Listen(listenMaxConnection)
		if err != nil {
			return nil, err
		}

		return &TCPListener{
			socket: s,
		}, nil
	case "udp":
		addr, err := netip.ParseAddrPort(externalAddress)
		if err != nil {
			return nil, err
		}
		s := core.CreateUDPSocket()
		s.Bind(addr)

		return &UDPListener{
			socket: s,
		}, nil
	}

	return nil, nil
}

func (l *TCPListener) Close() error {
	err := l.socket.Close()
	if err != nil {
		return err
	}
	return nil
}

func (l *TCPListener) Fd() int32 {
	return l.socket.Fd
}

func (l *UDPListener) Close() error {
	err := l.socket.Close()
	if err != nil {
		return err
	}
	return nil
}

func (l *UDPListener) Fd() int32 {
	return l.socket.Fd
}
