package main

import (
	"context"
	"github.com/touka-aoi/low-level-server/interface/server"
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
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	strAddress := "127.0.0.1:8081"
	ip, err := netip.ParseAddrPort(strAddress)
	if err != nil {
		slog.DebugContext(ctx, "ParseAddrPort", "err", err)
	}

	s := server.NewAcceptor()
	defer s.Close()
	err = s.Listen(strAddress)
	if err != nil {
		slog.DebugContext(ctx, "Listen", "err", err)
	}
	slog.InfoContext(ctx, "Server Start", "address", ip.Addr().String(), "port", ip.Port())
	s.Serve(ctx)

	//<-ctx.Done()

	//fdMax := 1 << 20

}
