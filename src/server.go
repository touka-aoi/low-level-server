package server

import (
	"encoding/binary"
	"log"
	"net/netip"
	"syscall"
	"unsafe"
)

const maxConnection = 4096

type sockAddr struct {
	Family uint16
	Data   [14]byte
}

func Listen(address string) error {
	addr, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}
	socket := CreateSocket()
	socket.Bind(addr)
	socket.Listen(maxConnection)

	return nil
}

type Socket struct {
	fd int
}

func CreateSocket() *Socket {
	// システムコールを叩いてSocketを作る
	// ノンブロッキング、プロセス継承なしでよさそう、tcpを選びたいので、SOCK_STREAMの0番でいいのかな
	// 基本的にひとつのソケットタイプには一つのプロトコルが割り当てられる
	// AF_INET | SOCK_STREAM なので 0でいいはず
	// https://man7.org/linux/man-pages/man2/socket.2.html
	fd, _, errno := syscall.Syscall6(
		syscall.SYS_SOCKET,
		syscall.AF_INET,
		syscall.SOCK_STREAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
		0,
		0,
		0,
		0)

	if fd < 0 {
		log.Printf("System call failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}

	return &Socket{fd: int(fd)}
}

func (s *Socket) Bind(address netip.AddrPort) {
	// https://man7.org/linux/man-pages/man2/bind.2.html
	sockaddr := sockAddr{
		Family: syscall.AF_INET,
	}

	// port
	port := address.Port()
	binary.BigEndian.PutUint16(sockaddr.Data[:], port)

	// address
	addr := address.Addr().AsSlice()
	for i := 0; i < len(addr); i++ {
		sockaddr.Data[2+i] = addr[i]
	}

	res, _, errno := syscall.Syscall6(
		syscall.SYS_BIND,
		uintptr(s.fd),
		uintptr(unsafe.Pointer(&sockaddr)),
		uintptr(unsafe.Sizeof(&sockaddr)),
		0,
		0,
		0)

	if res < 1 {
		log.Printf("Bind failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}
}

func (s *Socket) Listen(maxConn int) {
	res, _, errno := syscall.Syscall6(
		syscall.SYS_LISTEN,
		uintptr(maxConn),
		0,
		0, 0,
		0,
		0)

	if res < 1 {
		log.Printf("Listen failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}
}

// クローズ処理？
func (s *Socket) Unbind() {

}

func Accept() {

}

func Serve() {

}
