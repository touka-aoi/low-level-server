package server

import (
	"context"
	"encoding/binary"
	"golang.org/x/sys/unix"
	"log"
	"net/netip"
	"unsafe"
)

const maxConnection = 4096

type sockAddr struct {
	Family uint16
	Data   [14]byte
}

type UringParams struct {
	SqEntry       uint32
	CqEntry       uint32
	Flags         uint32
	SqThreadCPU   uint32
	SqThreadIdle  uint32
	Features      uint32
	WqFd          uint32
	Resv          [3]uint32
	SQRingOffsets SQRingOffsets
	CQRingOffsets CQRingOffsets
}

type SQRingOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Flags       uint32
	Dropped     uint32
	Array       uint32
	Resv1       uint32
	UserAddr    uint64
}

type CQRingOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Overflow    uint32
	CQEs        uint32
	Flags       uint32
	Resv1       uint32
	UserAddr    uint64
}

func CreateUring(entries uint32) {
	params := UringParams{}
	_, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_SETUP,
		uintptr(entries),
		uintptr(unsafe.Pointer(&params)),
		0,
		0,
		0,
		0)

	if errno != 0 {
		log.Printf("CreateUring failed: %v", errno)
	}

}

func Listen(ctx context.Context, address string) error {
	addr, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}
	socket := CreateSocket()
	socket.Bind(addr)
	socket.Listen(maxConnection)

	select {
	case <-ctx.Done():
		return nil
	default:
	}

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
	fd, _, errno := unix.Syscall6(
		unix.SYS_SOCKET,
		unix.AF_INET,
		unix.SOCK_STREAM|unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC,
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
	res, _, errno := unix.Syscall6(
		unix.SYS_LISTEN,
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
	CreateUring(maxConnection)
}

func Serve() {

}
