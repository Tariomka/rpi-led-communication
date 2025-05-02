package tcp

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/Tariomka/led-common-lib/pkg/network"
	"github.com/Tariomka/rpi-led-communication/internal/common"
)

type ServerConfig struct {
	Listener net.Listener
	Logger   *slog.Logger
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
	if config.Listener == nil {
		return nil, common.ErrNoListener
	}

	if config.Logger == nil {
		config.Logger = common.NewConsoleLogger(slog.LevelDebug)
	}

	return &LedServer{
		listener:  config.Listener,
		logger:    config.Logger,
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
		ls.logger.Debug(
			"new connection aquired:",
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
	ls.broadcast(network.NewMessagePacket(message))
}

func (ls *LedServer) receive(connection *Connection) {
	defer ls.removeConnection(connection)
	for {
		packet, err := connection.ReadPacket()
		if err != nil {
			switch err {
			case io.EOF:
				ls.logger.Info(
					"user disconnected:",
					"connection", connection.connection.RemoteAddr())
			default:
				ls.logger.Error("failed to read data from connection:", "error", err)
			}
			break
		}

		ls.logger.Debug(
			"received data:",
			"version", packet.Version,
			"type", packet.Type,
			"data", string(packet.Data))
	}
}

func (ls *LedServer) broadcast(packet network.Packet) {
	for connection := range ls.conns.Range {
		connection.(*Connection).WritePacket(packet)
	}
}

func (ls *LedServer) removeConnection(conn *Connection) {
	conn.Close()
	ls.conns.Delete(conn)
}
