package live

import "log/slog"

type LiveHandler struct {
}

func NewLiveHandler() *LiveHandler {
	return &LiveHandler{}
}

func (l *LiveHandler) JoinRoom() {
	slog.Debug("JoinRoom")
}

func (l *LiveHandler) LeaveRoom() {
	slog.Debug("LeaveRoom")
}

func (l *LiveHandler) Connect() {
	slog.Debug("Connect")
}

func (l *LiveHandler) Disconnect() {
	slog.Debug("Disconnect")
}

func (l *LiveHandler) ReceiveData() {
	slog.Debug("ReceiveData")
}

func (l *LiveHandler) SendData() {
	slog.Debug("SendData")
}

func (l *LiveHandler) SendControl() {
	slog.Debug("SendControl")
}

func (l *LiveHandler) SendHeartbeat() {
	slog.Debug("SendHeartbeat")
}

func (l *LiveHandler) ReceiveControl() {
	slog.Debug("ReceiveControl")
}

func (l *LiveHandler) ReceiveHeartbeat() {
	slog.Debug("ReceiveHeartbeat")
}
