package server

import (
	"net"
	"sync"
	"time"

	"github.com/Tariomka/led-common-lib/pkg/communication/tcp"
	"github.com/soypat/seqs"
	"github.com/soypat/seqs/stacks"
)

type connection struct {
	net.Conn
	sock     *stacks.TCPConn // nil when Conn is not a seqs socket
	remote   string
	closed   chan struct{} // closed when the connection is closed
	finished chan struct{} // closed when the main loop has handled the disconnect
	once     sync.Once
}

func newConnection(conn net.Conn) *connection {
	sock, _ := conn.(*stacks.TCPConn)
	return &connection{
		Conn:     conn,
		sock:     sock,
		remote:   conn.RemoteAddr().String(),
		closed:   make(chan struct{}),
		finished: make(chan struct{}),
	}
}

func (this *connection) close() {
	this.once.Do(func() {
		if this.owned() {
			this.Conn.Close()
		}
		close(this.closed)
	})
}

// owned reports whether the socket still carries this connection. seqs recycles a socket in place
// once the peer resets it; closing it then would kill the socket for good or close the next
// desktop's connection.
func (this *connection) owned() bool {
	return this.sock == nil || this.inState(seqs.StateEstablished) || this.inState(seqs.StateCloseWait)
}

func (this *connection) inState(state seqs.State) bool {
	return this.sock.State() == state && this.sock.RemoteAddr().String() == this.remote
}

// waitReadable returns once data is buffered or the connection is no longer established. A blocked
// seqs Read polls with a back-off that grows to 500 ms, which would delay every desktop message by
// that much.
func (this *connection) waitReadable() {
	for this.sock != nil && this.sock.BufferedInput() == 0 && this.inState(seqs.StateEstablished) {
		time.Sleep(time.Millisecond)
	}
}

// read receives one message; a recycled socket may already hold the next desktop's data.
func (this *connection) read(buffer []byte) (tcp.Message, error) {
	if !this.owned() {
		return tcp.Message{}, net.ErrClosed
	}
	return tcp.Read(this.Conn, buffer)
}
