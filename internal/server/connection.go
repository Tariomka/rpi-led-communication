package server

import (
	"io"
	"net/netip"
	"sync"
	"time"
)

type IConnection interface {
	io.ReadWriteCloser
	SetReadDeadline(t time.Time) error
	RemoteAddrPort() netip.AddrPort
}

type connection struct {
	IConnection
	remote   string
	closed   chan struct{} // closed when the connection is closed
	finished chan struct{} // closed when the main loop has handled the disconnect
	once     sync.Once
}

func newConnection(conn IConnection) *connection {
	return &connection{
		IConnection: conn,
		remote:      conn.RemoteAddrPort().String(),
		closed:      make(chan struct{}),
		finished:    make(chan struct{}),
	}
}

// close starts the TCP close and wakes a blocked reader; the pool aborts the connection if the
// peer never answers.
func (this *connection) close() {
	this.once.Do(func() {
		this.IConnection.Close()
		this.IConnection.SetReadDeadline(time.Now())
		close(this.closed)
	})
}
