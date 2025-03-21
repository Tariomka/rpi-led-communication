package tcp

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/Tariomka/rpi-led-communication/internal/common"
)

type ServerConfig struct {
	Address  string
	Listener net.Listener
	Logger   *slog.Logger
}

func NewConfig() ServerConfig {
	return ServerConfig{
		Address: ":42069",
	}
}

type Server interface {
	Start()
	Stop()
	Send(message string)
}

type LedServer struct {
	listener net.Listener
	logger   *slog.Logger

	waitGroup *sync.WaitGroup
	conns     sync.Map
}

func NewServer(config ServerConfig) (Server, error) {
	var (
		listener net.Listener
		logger   *slog.Logger
		err      error
	)

	listener = config.Listener
	if listener == nil {
		listener, err = net.Listen("tcp", config.Address)
		if err != nil {
			return nil, err
		}
	}

	logger = config.Logger
	if logger == nil {
		logger = slog.New(common.NewLogHandler(
			func(message string) { fmt.Println(message) },
			&slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return &LedServer{
		listener:  listener,
		logger:    logger,
		waitGroup: &sync.WaitGroup{},
	}, nil
}

func (ls *LedServer) Start() {
	ls.logger.Debug("Starting up server")
	fmt.Println()
	for {
		connection, err := ls.listener.Accept()
		if err != nil {
			ls.logger.Error("failed to accept connection:", "error", err)
			break
		}

		connWrapper := NewConnection(connection, ls.waitGroup)
		ls.conns.Store(connWrapper, true)
		ls.logger.Debug("new connection aquired:",
			"connection", connWrapper.connection.RemoteAddr())

		go ls.receive(connWrapper)
	}
	ls.waitGroup.Wait()
}

func (ls *LedServer) Stop() {
	for connection := range ls.conns.Range {
		connection.(*Connection).Close()
	}
	ls.listener.Close()
}

func (ls *LedServer) Send(message string) {
	ls.broadcast(NewMessagePacket(message))
}

func (ls *LedServer) receive(connection *Connection) {
	defer ls.removeConnection(connection)
	for {
		packet, err := connection.ReadPacket()
		if err != nil {
			switch err {
			case io.EOF:
				ls.logger.Info("user disconnected:", "connection", connection.connection.RemoteAddr())
			default:
				ls.logger.Error("failed to read data from connection:", "error", err)
			}
			break
		}

		ls.logger.Debug("received data:",
			"version", packet.Version,
			"type", packet.Type,
			"data", string(packet.Data))
	}
}

func (ls *LedServer) broadcast(packet Packet) {
	for connection := range ls.conns.Range {
		connection.(*Connection).WritePacket(packet)
	}
}

func (ls *LedServer) removeConnection(conn *Connection) {
	conn.Close()
	ls.conns.Delete(conn)
}
