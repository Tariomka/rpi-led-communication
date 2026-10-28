package server

import (
	"errors"
	"io"
	"log/slog"

	"github.com/Tariomka/led-common-lib/pkg/communication/tcp"
	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/Tariomka/rpi-led-communication/internal/component"
)

const outboxSize = 256 // fits the Hello and the whole line history sent on connect

type RequestKind int

const (
	Connected RequestKind = iota
	Disconnected
	Received
)

// Request is a desktop event for the main loop; Message is valid until Done is called.
type Request struct {
	Kind    RequestKind
	Message tcp.Message
}

// Server serves one desktop at a time; a new connection replaces the current one.
type Server struct {
	listener *component.Listener
	logger   *slog.Logger
	current  *connection
	buffer   []byte

	requests chan Request
	handled  chan struct{}
	outbox   chan tcp.Message
}

func NewServer(listener *component.Listener, logger *slog.Logger) (*Server, error) {
	if listener == nil {
		return nil, common.ErrNoListener
	}

	if logger == nil {
		logger = common.NewNoopLogger()
	}
	return &Server{
		listener: listener,
		logger:   logger,
		buffer:   make([]byte, 4+tcp.MaxChunk), // the largest payload: offset u32 + chunk
		requests: make(chan Request),
		handled:  make(chan struct{}, 1),
		outbox:   make(chan tcp.Message, outboxSize),
	}, nil
}

// Requests delivers desktop events; call Done after handling each one.
func (this *Server) Requests() <-chan Request { return this.requests }

func (this *Server) Done() { this.handled <- struct{}{} }

// Send queues a message for the desktop without blocking; it is dropped when the queue is full.
func (this *Server) Send(message tcp.Message) {
	select {
	case this.outbox <- message:
	default:
	}
}

// Start accepts connections until the listener fails.
func (this *Server) Start() {
	defer component.Recover("server")
	this.logger.Debug("Starting up server")
	for {
		conn, err := this.listener.Accept()
		if err != nil {
			this.logger.Error("Failed to accept connection", "error", err)
			return
		}

		if this.current != nil {
			this.logger.Info("Replacing the desktop connection")
			this.current.close()
			<-this.current.finished
		}
		for len(this.outbox) > 0 {
			<-this.outbox
		}

		this.current = newConnection(conn)
		go this.serve(this.current)
	}
}

func (this *Server) Stop() {
	if this.current != nil {
		this.current.close()
	}
	this.listener.Close()
}

func (this *Server) serve(conn *connection) {
	defer component.Recover("server reader")
	this.logger.Info("Desktop connected", "address", conn.remote)

	go this.write(conn)
	this.request(Request{Kind: Connected})
	for {
		message, err := tcp.Read(conn, this.buffer)
		switch {
		case err == nil:
			this.request(Request{Kind: Received, Message: message})
		case errors.Is(err, common.ErrUnsupportedVersion):
			this.logger.Warn("Unsupported protocol version", "version", message.Version, "type", message.Type)
		case errors.Is(err, common.ErrPayloadTooLarge):
			this.logger.Warn("Message too large, skipped", "type", message.Type)
		default:
			if err == io.EOF {
				this.logger.Info("Desktop disconnected", "address", conn.remote)
			} else {
				this.logger.Error("Desktop connection failed", "address", conn.remote, "error", err)
			}
			conn.close()
			this.request(Request{Kind: Disconnected})
			close(conn.finished)
			return
		}
	}
}

func (this *Server) write(conn *connection) {
	defer component.Recover("server writer")
	for {
		select {
		case <-conn.closed:
			return
		default:
		}

		select {
		case message := <-this.outbox:
			if tcp.Write(conn, message) != nil {
				conn.close()
				return
			}
		case <-conn.closed:
			return
		}
	}
}

func (this *Server) request(request Request) {
	this.requests <- request
	<-this.handled
}
