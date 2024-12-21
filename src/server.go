package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
)

const maxConnection = 4096

type Server struct {
	connections []chan Peer
	listener    net.Listener
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Listen(ctx context.Context, address string) error {
	listener, err := s.listenTCP4(ctx, address)
	if err != nil {
		return err
	}
	listener.Accept(ctx, maxConnection)

	return nil
}

func (s *Server) Serve() {
	nfd, err := s.listener.Accept()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(nfd)
}

type Peer struct {
	Fd int32
	Ip netip.AddrPort
}

func (s *Server) listenTCP4(ctx context.Context, address string) (*Socket, error) {
	addr, err := netip.ParseAddrPort(address)
	if err != nil {
		return nil, err
	}
	socket := CreateSocket()

	socket.Bind(addr)
	socket.Listen(maxConnection)

	return socket, nil
}

func Serve() {

}
