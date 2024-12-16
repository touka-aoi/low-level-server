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

	addr, _ := netip.ParseAddrPort("127.0.0.1:8080")
	socket := server.CreateSocket()
	socket.Bind(addr)
	socket.Listen(ctx, 4096)
	slog.Debug("Start Listen")

	go server.Accept(ctx, socket)

	<-ctx.Done()

	//_, _, err := unix.Accept4(int(socket.Fd), unix.SOCK_NONBLOCK)
	//if err != nil {
	//	panic(err)
	//}

	//fmt.Println("accept")
	//sockaddr := sockAddr{}
	//addrLen := uint32(unsafe.Sizeof(sockaddr))
	//fd, _, errno := unix.Syscall6(
	//	unix.SYS_ACCEPT4,
	//	uintptr(socket.Fd),
	//	uintptr(unsafe.Pointer(&sockaddr)),
	//	uintptr(unsafe.Pointer(&addrLen)),
	//	0,
	//	0,
	//	0,
	//)

	//if errno != 0 {
	//	log.Printf("Accept failed with errno: %d (%s)", errno, errno.Error())
	//	panic(unix.ErrnoName(errno))
	//}
	//
	//fmt.Println("Accpet", fd)

	//server.Accept(ctx, socket)
	//if err != nil {
	//	log.Fatal(err)
	//}

}
