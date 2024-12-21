package infra

type Socket struct {
	Fd int32
}

func (s *Socket) Accept() (net.Conn, error) {

}

func (s *Socket) Close() error {
	// fdを閉じる
}

func (s *Socket) Addr() net.Addr {

}
