//go:build linux

package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/touka-aoi/low-level-server/handler"
	"github.com/touka-aoi/low-level-server/core/engine"
)

func main() {
	// ログレベルをDEBUGに設定
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// コンテキストの設定
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// シグナルハンドリング
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		slog.Info("Shutting down server...")
		cancel()
	}()

	// リスナーの作成
	listener, err := engine.Listen("tcp", "127.0.0.1:8080", 128)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	slog.Info("Server starting", "address", "127.0.0.1:8080")

	// エンジンの作成
	netEngine := engine.NewUringNetEngine()
	defer netEngine.Close()

	// Accept操作を開始
	if err := netEngine.Accept(ctx, listener); err != nil {
		log.Fatalf("Failed to start accepting connections: %v", err)
	}

	// セッションマネージャーの作成
	sessionManager := handler.NewSessionManager(netEngine)

	// サーバーの開始
	slog.Info("Server ready to accept connections")
	sessionManager.Serve(ctx)
}