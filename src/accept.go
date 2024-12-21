package server

import (
	"context"
	"log/slog"
)

type Acceptor struct {
	uring      *Uring
	socket     *Socket
	AcceptChan chan Peer
	sockAddr   sockAddr
	sockLen    uint32
}

// TODO: 名前を変える
func NewAcceptor() *Acceptor {
	// ここでsocketを作っちゃう
	return &Acceptor{}
}

func (a *Acceptor) Listen(ctx context.Context, address string) error {
	// Listenはこここ
}

func (a *Acceptor) Accept(ctx context.Context) {
	for peer := range a.AcceptChan {
		// io-uring で read する待機列に入れる
		a.watchPeer(&peer)
	}
}

func (a *Acceptor) watchPeer(peer *Peer) {
	err := a.uring.WatchRead(peer)
	if err != nil {
		slog.Debug("WatchRead", "err", err)
	}
}

func (a *Acceptor) MainLoop(ctx context.Context) {
	// ノンブロッキングでreadできないかどうかを監視する
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		}
		//TODO: cqeからデータを受け取ることがわかるもっといい名前
		a.uring.Wait()
		//data := a.uring.Read()
		//a.service.HandleData(data)
	}

	slog.InfoContext(ctx, "MainLoop end")
}
