package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/touka-aoi/low-level-server/application/live"
	"github.com/touka-aoi/low-level-server/core/engine"
	"github.com/touka-aoi/low-level-server/server"
	"github.com/touka-aoi/low-level-server/transport/streaming"
)

const (
	protocol = "tcp"
)

func main() {
	// Parse flags
	var (
		address = flag.String("address", "127.0.0.1", "Host to listen on")
		port    = flag.Int("port", 8080, "Port to listen on")
	)
	flag.Parse()

	// Setup logging
	logLevel := slog.LevelDebug

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	netEngine := engine.NewUringNetEngine()
	defer func() {
		err := netEngine.Close()
		if err != nil {
			slog.Error("Failed netEngine.Close()", "error", err)
		}
	}()

	config := server.NetworkServerConfig{
		Protocol: protocol,
		Address:  *address,
		Port:     *port,
	}

	realTimeHandler := live.NewLiveHandler()
	realTimeApp := streaming.NewLiveStreaming()
	realTimeApp.SetHandler(realTimeHandler)

	networkServer := server.NewNetworkServer(netEngine, config, nil, realTimeApp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := networkServer.Listen(ctx)
	if err != nil {
		slog.Error("Failed networkServer.Listen()", "error", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutdown signal received")
		cancel()
	}()

	networkServer.Serve(ctx)

	slog.Info("Server stopped")
}
