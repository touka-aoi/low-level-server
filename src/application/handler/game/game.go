package game

import (
	"context"
	"github.com/touka-aoi/low-level-server/gen/proto"
)

type LogicServer interface {
	Tick(ctx context.Context) (interface{}, error)
	PostEvent(event interface{}) error
}

type Player interface {
	Send([]byte) error
}

type GameServer struct {
	logicServer LogicServer
	player      []Player
}

func (g GameServer) dataHandler(h func(ctx context.Context, data []byte) (interface{}, error)) func(ctx context.Context, data []byte) (interface{}, error) {
	return func(ctx context.Context, data []byte) (interface{}, error) {
		_, _ = h(ctx, data)
		if event, ok := data.(*proto.PlayerAction); ok {
			// ここで型判定してるんだから型で送った方が効率的
			err := g.logicServer.PostEvent(event)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
}

//
//func (g *GameServer) LoginHandler() {
//	// ログイン処理
//}
//
//func (g *GameServer) AuthHandler() {
//	// 認証処理
//}
//
//func (g *GameServer) dataHandler(h func(ctx context.Context, data []byte)) func(ctx context.Context, data []byte) {
//	return h
//}

//func (g *GameServer) DataHandler(ctx context.Context) error {
//	res := g.dataHandler(ctx)
//
//	switch reslst := results.(type) {
//	case []string:
//		// ジェネリック的処理するか
//		g.logicServer.PostEvent < GRPCHogeEvent > (event)
//	case []int:
//		// 何か処理
//	default:
//		// 何か処理
//	}
//}
//
//func (g *GameServer) Tick(ctx context.Context) {
//	// 30fpsで処理する
//	result, err := s.logixServer.Tick(ctx)
//	if err != nil {
//		// 1フレーム落としたらどうするかは後で考える
//	}
//	for _, user := range g.users {
//		user.Send(result)
//	}
//}
