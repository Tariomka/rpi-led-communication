package tcp

import (
	"fmt"

	"github.com/Tariomka/rpi-led-communication/internal/common"
)

type DataType byte

const (
	Message DataType = iota
	RGB8x8
	RGB16x16
	RGB8x32
	Mono8x8
)

const (
	headerSize      = 2
	byteSize8x8     = 192
	byteSize8x8Mono = 64
	byteSize16x16   = 768
	byteSize8x32    = 768
)

type Packet struct {
	Version byte
	Type    DataType
	Data    []byte
}

func NewMessagePacket(data string) Packet {
	return Packet{
		Version: 1,
		Type:    Message,
		Data:    []byte(data),
	}
}

func NewLedPacket(dtype DataType, data []byte) Packet {
	return Packet{
		Version: 1,
		Type:    dtype,
		Data:    data,
	}
}

func (p Packet) Marshall() []byte {
	bytes := make([]byte, 0, 1+1+len(p.Data))
	bytes = append(bytes, p.Version, byte(p.Type))
	bytes = append(bytes, p.Data...)
	return bytes
}

func UnmarshallPacket(data []byte) (*Packet, error) {
	if len(data) < headerSize {
		return nil, fmt.Errorf("data too small")
	}

	version := data[0]
	if version != 1 {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	dtype := DataType(data[1])

	packet := &Packet{
		Version: version,
		Type:    dtype,
	}

	switch dtype {
	case RGB8x8:
		packet.Data = data[headerSize : byteSize8x8+headerSize]
	case RGB16x16:
		packet.Data = data[headerSize : byteSize16x16+headerSize]
	case RGB8x32:
		packet.Data = data[headerSize : byteSize8x32+headerSize]
	case Mono8x8:
		packet.Data = data[headerSize : byteSize8x8Mono+headerSize]
	case Message:
		if end := common.FindFirstIndex(data[headerSize:], 0); end > -1 {
			packet.Data = data[headerSize : end+headerSize]
		} else {
			packet.Data = data[headerSize:]
		}
	default:
		packet.Data = data[headerSize:]
	}

	return packet, nil
}
