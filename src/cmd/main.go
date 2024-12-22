package main

import (
	"log/slog"
	"os"
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
	//ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	//defer cancel()

	//s := server.NewAcceptor()
	//defer s.Close()
	//err := s.Listen("127.0.0.1:8083")
	//if err != nil {
	//	slog.DebugContext(ctx, "Listen", "err", err)
	//}
	//slog.InfoContext(ctx, "Server Start")
	//s.Serve(ctx)

	//<-ctx.Done()

	//fdMax := 1 << 20

}
