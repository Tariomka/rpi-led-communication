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
	waitGroup.Add(1)

	return &Connection{
		connection: connection,
		waitGroup:  waitGroup,
	}
}

func (c *Connection) ReadPacket() (*network.Packet, error) {
	// Add multiple packet reading?
	// read with buffer? how current implementation reads multiple packets?
	buffer := make([]byte, 1024)
	_, err := c.connection.Read(buffer)
	if err != nil {
		return nil, err
	}

	packet, err := network.UnmarshallPacket(buffer)
	if err != nil {
		return nil, err
	}

	return packet, nil
}

func (c *Connection) WritePacket(packet network.Packet) {
	// TODO: add more handling?
	c.connection.Write(packet.Marshall())
}

func (c *Connection) Close() {
	c.connection.Close()
	c.waitGroup.Done()
}
