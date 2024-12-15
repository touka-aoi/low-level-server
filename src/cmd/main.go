package main

import (
	"context"
	server "github.com/touka-aoi/low-level-server"
	"log/slog"
	"net/netip"
	"os"
	"os/signal"
)

type sockAddr struct {
	Family uint16
	Data   [14]byte
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // ログレベルをデバッグに設定
	}))
	slog.SetDefault(logger) // デフォルトロガーを設定
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	// ここは最終的には server.Run() とかにしたい

	addr, _ := netip.ParseAddrPort("127.0.0.1:8000")
	socket := server.CreateSocket()
	socket.Bind(addr)
	go socket.Listen(ctx, 4096)

	//var addrOut unix.SockaddrInet4
	//var addrLen uint32 = uint32(unsafe.Sizeof(addrOut))
	//_, _, errno := unix.Syscall6(
	//	unix.SYS_ACCEPT4,
	//	uintptr(socket.Fd),
	//	uintptr(unsafe.Pointer(&addrOut)),
	//	uintptr(unsafe.Pointer(&addrLen)),
	//	0,
	//	0,
	//	0,
	//)
	//
	//if errno != 0 {
	//	log.Printf("Listen failed with errno: %d (%s)", errno, errno.Error())
	//	panic(errno)
	//}

	select {
	case <-ctx.Done():
	}
	//server.Accept(ctx, socket)
	//if err != nil {
	//	log.Fatal(err)
	//}

}
