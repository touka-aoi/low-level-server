package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"net/netip"
	"unsafe"
)

const maxConnection = 4096

type Server struct {
	connections []chan Peer
	listener    net.Listener
}

func NewServer() *Server {
	return &Server{}
}

//func hoge() {
//	fd, err := net.Listen("tcp", "hogehoge")
//	if err != nil {
//		log.Fatal(err)
//	}
//	nfd, err := fd.Accept()
//	if err != nil {
//		log.Fatal(err)
//	}
//}

func (s *Server) Listen(ctx context.Context, address string) error {
	listener, err := s.listenTCP4(ctx, address)
	if err != nil {
		return err
	}

	s.listener = listener

	return nil
}

func (s *Server) Serve() {
	nfd, err := s.listener.Accept()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(nfd)
}

type sockAddr struct {
	Family uint16
	Data   [14]byte
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
	socket.Listen(ctx, maxConnection)

	return socket, nil
}

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

func CreateSocket() *Socket {
	// システムコールを叩いてSocketを作る
	// ノンブロッキング、プロセス継承なしでよさそう、tcpを選びたいので、SOCK_STREAMの0番でいいのかな
	// 基本的にひとつのソケットタイプには一つのプロトコルが割り当てられる
	// AF_INET | SOCK_STREAM なので 0でいいはず
	// https://man7.org/linux/man-pages/man2/socket.2.html
	fd, _, errno := unix.Syscall6(
		unix.SYS_SOCKET,
		unix.AF_INET,
		unix.SOCK_STREAM|unix.SOCK_CLOEXEC,
		0,
		0,
		0,
		0)

	if fd < 0 {
		log.Printf("Socket failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}

	return &Socket{Fd: int32(fd)}
}

func (s *Socket) Bind(address netip.AddrPort) {
	// https://man7.org/linux/man-pages/man2/bind.2.html
	sockaddr := sockAddr{
		Family: unix.AF_INET,
	}

	// port
	port := address.Port()
	binary.BigEndian.PutUint16(sockaddr.Data[:], port)

	// address
	addr := address.Addr().AsSlice()
	for i := 0; i < len(addr); i++ {
		sockaddr.Data[2+i] = addr[i]
	}

	res, _, errno := unix.Syscall6(
		unix.SYS_BIND,
		uintptr(s.Fd),
		uintptr(unsafe.Pointer(&sockaddr)),
		uintptr(unsafe.Sizeof(sockaddr)),
		0,
		0,
		0)

	if res != 0 {
		log.Printf("Bind failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}
}

func (s *Socket) Listen(ctx context.Context, maxConn int) {
	res, _, errno := unix.Syscall6(
		unix.SYS_LISTEN,
		uintptr(s.Fd),
		uintptr(maxConn),
		0,
		0,
		0,
		0)

	if res != 0 {
		log.Printf("Listen failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}
}

// クローズ処理？
func (s *Socket) Unbind() {

}

func Accept(ctx context.Context, socket *Socket) {
	uring := CreateUring(maxConnection)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			uring.Accpet(socket)
		}
	}
	//TODO 受信したものをServeへ回す
}

func Serve() {

}
