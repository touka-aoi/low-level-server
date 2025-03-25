package main

import (
	gen "github.com/touka-aoi/low-level-server/gen/proto"
	"google.golang.org/protobuf/encoding/protodelim"
	"os"
)

// check proto marshal result
func main() {
	msg := &gen.ActionResult{}
	msg.SetSuccess(true)
	msg.SetMessage("Hello, World!")
	file, err := os.Create("output.bin")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = protodelim.MarshalTo(file, msg)
	if err != nil {
		panic(err)
	}
}
