package server

import "net/http"

// Acceptor is a struct that holds the server's socket and uring.
func hoge() {
	server := http.Server{}
	server.ListenAndServe()
}
