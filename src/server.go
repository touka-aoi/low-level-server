package server

import (
	"fmt"
	"syscall"
)

func Listen() {
	socket := CreateSocket()

	fmt.Println(socket)
}

type Socket struct {
	fd int
}

func CreateSocket() *Socket {
	// システムコールを叩いてSocketを作る
	// ノンブロッキング、プロセス継承なしでよさそう、tcpを選びたいので、SOCK_STREAMの0番でいいのかな
	// 基本的にひとつのソケットタイプには一つのプロトコルが割り当てられる
	// AF_INET | SOCK_STREAM なので 0でいいはず
	fd, _, err := syscall.Syscall6(
		syscall.SYS_SOCKET,
		syscall.AF_INET,
		syscall.SOCK_STREAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
		0,
		0,
		0,
		0)

	if fd < 0 {
		panic(err)
	}

	return &Socket{fd: int(fd)}
}

func (s *Socket) Bind() {

}

func (s *Socket) Listen() {

}

// クローズ処理？
func (s *Socket) Unbind() {

}

func Accept() {

}

func Serve() {

}
