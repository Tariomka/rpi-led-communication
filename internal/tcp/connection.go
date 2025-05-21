package tcp

import (
	"net"
	"sync"

	"github.com/Tariomka/led-common-lib/pkg/network"
)

type Connection struct {
	connection net.Conn
	waitGroup  *sync.WaitGroup
}

func NewConnection(connection net.Conn, waitGroup *sync.WaitGroup) *Connection {
	if connection == nil || waitGroup == nil {
		return nil
	}

	waitGroup.Add(1)
	return &Connection{
		connection: connection,
		waitGroup:  waitGroup,
	}
}

func (this *Connection) ReadPacket() (*network.Packet, error) {
	// Assuming that all data fits into a single packet potentialy might cause problems
	// in the future but it's fine for now, so until this becomes a brickwall, no need to change it
	buffer := make([]byte, 1024)
	_, err := this.connection.Read(buffer)
	if err != nil {
		return nil, err
	}

	packet, err := network.UnmarshallPacket(buffer)
	if err != nil {
		return nil, err
	}

	return packet, nil
}

func (this *Connection) WritePacket(packet network.Packet) {
	// TODO: add more handling?
	this.connection.Write(packet.Marshall())
}

func (this *Connection) Close() {
	this.connection.Close()
	this.waitGroup.Done()
}
