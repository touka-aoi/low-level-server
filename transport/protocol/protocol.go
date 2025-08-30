package protocol

import (
	"encoding/binary"
	"errors"
)

// HEADER LAYOUT
// +--------+--------+--------+--------+--------+--------+
// | Magic(2 bytes) | Type(1)|   Length(4 bytes)       |
// +--------+--------+--------+--------+--------+--------+
// | 0x61 0x6F      | 0x01   |  0x00 0x00 0x10 0x00    |
// +--------+--------+--------+--------+--------+--------+

const (
	MagicNumber uint16 = 0x616F
	HeaderSize         = 7

	TYPE_DATA      = 0x01
	TYPE_CONTROL   = 0x02
	TYPE_HEARTBEAT = 0x03

	CmdJoinRoom   uint8 = 0x01
	CmdLeaveRoom  uint8 = 0x02
	CmdConnect    uint8 = 0x03
	CmdDisconnect uint8 = 0x04
)

type LiveProtocol interface {
	JoinRoom()
	LeaveRoom()
	Connect()
	Disconnect()
	SendData()
	SendControl()
	SendHeartbeat()
	ReceiveData()
	ReceiveControl()
	ReceiveHeartbeat()
}

var (
	InsufficientData = errors.New("insufficient data")
	InvalidMagic     = errors.New("invalid magic")
	InCompleteData   = errors.New("incomplete data")
)

type Frame struct {
	Type    uint8
	Payload []byte
}

func ParseFrame(data []byte) (*Frame, error) {
	if len(data) < HeaderSize {
		return nil, InsufficientData
	}

	magic := binary.BigEndian.Uint16(data[0:2])
	if magic != MagicNumber {
		return nil, InvalidMagic
	}

	frameType := data[2]
	length := binary.BigEndian.Uint32(data[3:7])
	totalLength := HeaderSize + int(length)
	if len(data) < totalLength {
		return nil, InCompleteData
	}
	payload := data[HeaderSize:totalLength]

	return &Frame{
		Type:    frameType,
		Payload: payload,
	}, nil
}

func (f *Frame) Marshal() []byte {
	buf := make([]byte, HeaderSize+len(f.Payload))
	buf[2] = f.Type
	binary.BigEndian.PutUint32(buf[3:7], uint32(len(f.Payload)))
	// コピーしたくないけど方法を知らない
	copy(buf[HeaderSize:], f.Payload)
	return buf
}
