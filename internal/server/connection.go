package server

import (
	"net"
	"sync"
)

type connection struct {
	net.Conn
	closed   chan struct{} // closed when the connection is closed
	finished chan struct{} // closed when the main loop has handled the disconnect
	once     sync.Once
}

func newConnection(conn net.Conn) *connection {
	return &connection{
		Conn:     conn,
		closed:   make(chan struct{}),
		finished: make(chan struct{}),
	}
}

func (this *connection) close() {
	this.once.Do(func() {
		this.Conn.Close()
		close(this.closed)
	})
}
