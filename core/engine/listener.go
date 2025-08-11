//go:build linux

package engine

import (
	"net/netip"

	"github.com/touka-aoi/low-level-server/core/io"
)

type Listener interface {
	Fd() int32
	Close() error
}

type TCPListener struct {
	socket *io.Socket
}

type UDPListener struct {
	socket *io.Socket
}

func Listen(protocol, externalAddress string, listenMaxConnection int) (Listener, error) {
	switch protocol {
	case "tcp":
		addr, err := netip.ParseAddrPort(externalAddress)
		if err != nil {
			return nil, err
		}

		s := io.CreateTCPSocket()
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
		s := io.CreateUDPSocket()
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
