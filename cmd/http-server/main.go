package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/touka-aoi/low-level-server/application/http"
	"github.com/touka-aoi/low-level-server/core/engine"
	"github.com/touka-aoi/low-level-server/server"
)

func main() {
	// Parse flags
	var (
		host  = flag.String("host", "0.0.0.0", "Host to listen on")
		port  = flag.Int("port", 8080, "Port to listen on")
		debug = flag.Bool("debug", false, "Enable debug logging")
	)
	flag.Parse()

	// Setup logging
	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Create io_uring engine
	netEngine := engine.NewUringNetEngine()
	defer netEngine.Close()

	// Create HTTP application with default handlers
	router := http.DefaultHandlers()
	httpApp := http.NewHTTPApplication(router)

	// Create network server
	networkServer := server.NewNetworkServer(netEngine, httpApp, nil)

	// Create listener
	addr := fmt.Sprintf("%s:%d", *host, *port)
	listener, err := engine.Listen("tcp", addr, 128)
	if err != nil {
		slog.Error("Failed to create listener", "error", err)
		os.Exit(1)
	}
	defer listener.Close()

	slog.Info("HTTP server starting", "address", addr)

	// Start accepting connections
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := netEngine.Accept(ctx, listener); err != nil {
		slog.Error("Failed to start accepting connections", "error", err)
		os.Exit(1)
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutdown signal received")
		cancel()
	}()

	// Run the server
	networkServer.Serve(ctx)
	
	slog.Info("Server stopped")
}