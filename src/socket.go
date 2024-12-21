package server

import (
	"context"
	"encoding/binary"
	"log"
	"net/netip"
	"unsafe"

	"golang.org/x/sys/unix"
)

type sockAddr struct {
	Family uint16
	Data   [14]byte
}

type Socket struct {
	Fd        int32
	LocalAddr string
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

	port := address.Port()
	binary.BigEndian.PutUint16(sockaddr.Data[:], port)

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

	s.LocalAddr = address.String()
}

func (s *Socket) Listen(maxConn int) {
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

func (s *Socket) Accept(ctx context.Context, maxConnection int) *Socket {
	uring := CreateUring(uint32(maxConnection))
	go uring.Accpet(s)
	return nil
}

func (s *Socket) Close() {
	//TODO: unix syscall6 に変更する
	unix.Close(int(s.Fd))
}
