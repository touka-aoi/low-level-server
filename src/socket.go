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
	//TODO socket accpetの実装
	return nil
}

func (s *Socket) Close() error {
	res, _, errno := unix.Syscall6(unix.SYS_CLOSE, uintptr(s.Fd), 0, 0, 0, 0, 0)
	if res != 0 {
		//MEMO: touka-aoi errono型を返すのが正しいのか考えたい
		return errno
	}
	return nil
}
