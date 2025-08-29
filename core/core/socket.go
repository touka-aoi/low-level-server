//go:build linux

package core

import (
	"encoding/binary"
	"errors"
	"log/slog"
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

func CreateTCPSocket() *Socket {
	fd, _, errno := unix.Syscall6(
		unix.SYS_SOCKET,
		unix.AF_INET,
		unix.SOCK_STREAM|unix.SOCK_CLOEXEC,
		0,
		0,
		0,
		0)

	if fd < 0 {
		slog.Error("Failed to create socket", "errno", errno, "err", errno.Error())
		panic(errno)
	}

	opVal := int32(1)
	_, _, errno = unix.Syscall6(unix.SYS_SETSOCKOPT, fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, uintptr(unsafe.Pointer(&opVal)), unsafe.Sizeof(opVal), 0)
	if errno != 0 {
		slog.Error("Failed to set socket option", "errno", errno, "err", errno.Error())
		panic(errno)
	}

	return &Socket{Fd: int32(fd)}
}

func CreateUDPSocket() *Socket {
	fd, _, errno := unix.Syscall6(
		unix.SYS_SOCKET,
		unix.AF_INET,
		unix.SOCK_DGRAM|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK,
		0,
		0,
		0,
		0)

	if fd < 0 {
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
		panic(errno)
	}

	s.LocalAddr = address.String()
}

func (s *Socket) Listen(maxConn int) error {
	res, _, errno := unix.Syscall6(
		unix.SYS_LISTEN,
		uintptr(s.Fd),
		uintptr(maxConn),
		0,
		0,
		0,
		0)

	if res != 0 {
		slog.Error("Failed to listen", "errno", errno, "err", errno.Error())
		return errors.New(errno.Error())
	}

	return nil
}

func (s *Socket) Close() error {
	res, _, errno := unix.Syscall6(unix.SYS_CLOSE, uintptr(s.Fd), 0, 0, 0, 0, 0)
	if res != 0 {
		return errno
	}
	return nil
}
