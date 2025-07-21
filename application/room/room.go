package room

import (
	"context"
	"errors"
	"log/slog"
)

type Connection struct {
	ID string
}

type Logic interface {
	Run(ctx context.Context) error
	OnConnect(ctx context.Context, conn *Connection) error
	OnDisconnect(ctx context.Context, conn *Connection) error
}

type Room struct {
	ID          string
	Capacity    int
	Connections []*Connection
	Logic       Logic
}

type RoomManager struct {
	Rooms map[string]*Room
}

func NewRoomManager(size int) *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room, size),
	}
}

func (rm *RoomManager) CreateRoom(ctx context.Context, id string, capacity int, logic Logic) *Room {
	room := &Room{
		ID:          id,
		Capacity:    capacity,
		Connections: make([]*Connection, 0, capacity),
		Logic:       logic,
	}
	rm.Rooms[id] = room
	slog.DebugContext(ctx, "Room created", "roomID", id, "capacity", capacity)
	if logic != nil {
		go logic.Run(ctx)
		slog.DebugContext(ctx, "Room logic started", "roomID", id)
	}
	return room
}

func (rm *RoomManager) GetRoom(id string) (*Room, bool) {
	room, exists := rm.Rooms[id]
	return room, exists
}

func (rm *RoomManager) DeleteRoom(id string) {
	delete(rm.Rooms, id)
}

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
)

func (rm *RoomManager) AddConnection(ctx context.Context, roomID string, conn *Connection) error {
	room, exists := rm.Rooms[roomID]
	if !exists {
		return ErrRoomNotFound
	}
	err := room.JoinRoom(ctx, conn)
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, "Connection added to room", "roomID", roomID, "connID", conn.ID)
	return nil
}

func (rm *RoomManager) RemoveConnection(ctx context.Context, roomID string, connID string) error {
	room, exists := rm.Rooms[roomID]
	if !exists {
		return ErrRoomNotFound
	}

	room.LeaveRoom(ctx, connID)
	slog.DebugContext(ctx, "Connection removed from room", "roomID", roomID, "connID", connID)
	return nil
}

func (r *Room) IsFull() bool {
	return len(r.Connections) >= r.Capacity
}

func (r *Room) JoinRoom(ctx context.Context, conn *Connection) error {
	if r.IsFull() {
		return ErrRoomFull
	}
	r.Connections = append(r.Connections, conn)
	r.Logic.OnConnect(ctx, conn)
	return nil
}

func (r *Room) LeaveRoom(ctx context.Context, connID string) error {
	for i, c := range r.Connections {
		if c.ID == connID {
			r.Connections = append(r.Connections[:i], r.Connections[i+1:]...)
			r.Logic.OnDisconnect(ctx, c)
			return nil
		}
	}
	return errors.New("connection not found in room")
}

func (r *Room) FindConnection(connID string) (*Connection, bool) {
	for _, conn := range r.Connections {
		if conn.ID == connID {
			return conn, true
		}
	}
	return nil, false
}
