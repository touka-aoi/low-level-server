package internal

import "context"

type ServerConfig struct {
}

type Server2 struct {
	cfg      ServerConfig
	listener *Socket
}

func NewServer2() *Server2 {
	return &Server2{}
}

func (s *Server2) Listen(network, address string) error {
	s.listener, err := Socket.Listen(network, address)
	if err != nil {
		return err
	}
	return nil
}

// interfaceを満たしてたら入れれるようにしたらよさそう
func (s *Server2) AddCodec(c any) {
	s.ReadCodec = c
	s.WriteCodec = c
}

func (s *Server2) AddHandler(h any) {
	s.handler = h
}

func (s *Server2) Serve(ctx context.Context) {

}
