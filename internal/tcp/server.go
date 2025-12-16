package tcp

import (
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/Tariomka/led-common-lib/pkg/network"
	"github.com/Tariomka/rpi-led-communication/internal/common"
)

type Server interface {
	Start()
	Stop()
	Send(message string)
}

type LedServer struct {
	listener net.Listener
	logger   *slog.Logger

	waitGroup *sync.WaitGroup
	conns     sync.Map // Map of [*Connection]bool.
}

func NewServer(listener net.Listener, logger *slog.Logger) (Server, error) {
	if listener == nil {
		return nil, common.ErrNoListener
	}

	if logger == nil {
		logger = common.NewConsoleLogger(slog.LevelDebug)
	}
	return &LedServer{
		listener:  listener,
		logger:    logger,
		waitGroup: &sync.WaitGroup{},
	}, nil
}

func (this *LedServer) Start() {
	this.logger.Debug("Starting up server")
	this.logger.Debug("Main thread", "data", "Starting up server")
	for {
		this.logger.Debug("Main thread", "data", "Infinite loop: waiting for connection")
		connection, err := this.listener.Accept()
		if err != nil {
			this.logger.Error("Failed to accept connection:", "error", err)
			break
		}

		connWrapper := NewConnection(connection, this.waitGroup)
		if connWrapper == nil {
			continue
		}

		this.conns.Store(connWrapper, true)
		this.logger.Debug(
			"New connection aquired:",
			"connection", connWrapper.connection.RemoteAddr().String())

		go this.receive(connWrapper)
	}
	this.waitGroup.Wait()
}

func (this *LedServer) Stop() {
	for connection := range this.conns.Range {
		connection.(*Connection).Close()
	}
	this.listener.Close()
}

func (this *LedServer) Send(message string) {
	this.broadcast(network.NewMessagePacket(message))
}

// Must be a goroutine
func (this *LedServer) receive(connection *Connection) {
	defer this.removeConnection(connection)

	// When client disconnects, RemoteAddr() returns empty
	connAddress := connection.connection.RemoteAddr().String()

	this.logger.Debug("Goroutine 4", "data", "Once on connect: Receive started")
	for {
		this.logger.Debug("Goroutine 4", "data", "Infinite loop: Receive loop")
		packet, err := connection.ReadPacket()
		if err != nil {
			switch err {
			case io.EOF:
				this.logger.Info("User disconnected:", "connection", connAddress)
			default:
				this.logger.Error("Failed to read data from connection:", "error", err)
			}
			break
		}

		if packet.Version != network.V1 {
			this.logger.Error("Unsupported packet version", "packet version", packet.Version)
			connection.WritePacket(network.NewMessagePacket("Package is incorrect and was denied"))
			continue
		}

		if packet.Type == network.Message {
			this.logger.Info("Message from client received", "message", string(packet.Data))
			continue
		}

		// TODO: send to packet handler fucntion/class
		// TODO: send state to stm32 via UART
		this.logger.Debug(
			"Received data",
			"version", packet.Version,
			"type", packet.Type,
			"data", packet.Data)
		connection.WritePacket(network.NewMessagePacket("Packet received"))
	}
}

func (this *LedServer) broadcast(packet network.Packet) {
	for connection := range this.conns.Range {
		connection.(*Connection).WritePacket(packet)
	}
}

func (this *LedServer) removeConnection(conn *Connection) {
	conn.Close()
	this.conns.Delete(conn)
}
