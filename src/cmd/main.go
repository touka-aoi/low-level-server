package main

import (
	server "github.com/touka-aoi/low-level-server"
	"log"
)

type sockAddr struct {
	Family uint16
	Data   [14]byte
}

func main() {
	// ここは最終的には server.Run() とかにしたい
	err := server.Listen("127.0.0.1:8000")
	if err != nil {
		log.Fatal(err)
	}
}
