package main

import (
	"context"
	server "github.com/touka-aoi/low-level-server"
	"log"
	"os"
	"os/signal"
)

type sockAddr struct {
	Family uint16
	Data   [14]byte
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	// ここは最終的には server.Run() とかにしたい
	err := server.Listen(ctx, "127.0.0.1:8000")
	server.Accept()
	if err != nil {
		log.Fatal(err)
	}

}
