package runner

import (
	"errors"
	"log/slog"
	"machine"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/soypat/seqs"
	"github.com/soypat/seqs/stacks"
)

func testOut() {
	time.Sleep(2 * time.Second)
	config := NewConfig()
	ssid = config.SSID
	pass = config.Password
	logger := slog.New(slog.NewTextHandler(machine.USBCDC, &slog.HandlerOptions{
		Level: slog.LevelDebug - 2}))
	_, stack, _, err := SetupWithDHCP(SetupConfig{
		Hostname: "TCP-pico",
		Logger:   logger,
		TCPPorts: 1,
	})

	if err != nil {
		panic("in dhcp setup:" + err.Error())
	}

	// tcp
	tcpserver(stack)
}

// TCP listener test
func tcplistener(stack *stacks.PortStack, logger *slog.Logger) {
	const (
		tcpbufsize  = 512 // MTU - ethhdr - iphdr - tcphdr
		connTimeout = 5 * time.Second
		// Can help prevent stalling connections from blocking control the more connections you have.
		maxconns = 3
	)

	// Start TCP server.
	const listenPort = 1234
	listenAddr := netip.AddrPortFrom(stack.Addr(), listenPort)
	listener, err := stacks.NewTCPListener(stack, stacks.TCPListenerConfig{
		MaxConnections: maxconns,
		ConnTxBufSize:  tcpbufsize,
		ConnRxBufSize:  tcpbufsize,
	})
	if err != nil {
		panic("listener create:" + err.Error())
	}
	err = listener.StartListening(listenPort)
	if err != nil {
		panic("listener start:" + err.Error())
	}
	var buf [512]byte
	logger.Info("listening", slog.String("addr", listenAddr.String()))
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("listener accept:", slog.String("err", err.Error()))
			time.Sleep(time.Second)
			continue
		}
		logger.Info("new connection", slog.String("remote", conn.RemoteAddr().String()))
		err = conn.SetDeadline(time.Now().Add(connTimeout))
		if err != nil {
			logger.Error("conn set deadline:", slog.String("err", err.Error()))
			continue
		}
		for {
			n, err := conn.Read(buf[:])
			if err != nil {
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					logger.Error("conn read:", slog.String("err", err.Error()))
				}
				break
			}
			_, err = conn.Write(buf[:n])
			if err != nil {
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					logger.Error("conn write:", slog.String("err", err.Error()))
				}
				break
			}
		}
		err = conn.Close()
		if err != nil {
			logger.Error("conn close:", slog.String("err", err.Error()))
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TCP server test
func tcpserver(stack *stacks.PortStack) {
	// Start TCP server.
	const socketBuf = 1024
	const listenPort = 42069
	listenAddr := netip.AddrPortFrom(stack.Addr(), listenPort)
	socket, err := stacks.NewTCPConn(stack, stacks.TCPConnConfig{TxBufSize: socketBuf, RxBufSize: socketBuf})
	if err != nil {
		panic("socket create:" + err.Error())
	}
	println("start listening on:", listenAddr.String())
	err = foreverTCPListenEcho(socket, listenAddr)
	if err != nil {
		panic("socket listen:" + err.Error())
	}
}

func foreverTCPListenEcho(socket *stacks.TCPConn, addr netip.AddrPort) error {
	var iss seqs.Value = 100
	var buf [512]byte
	for {
		iss += 200
		err := socket.OpenListenTCP(addr.Port(), iss)
		if err != nil {
			return err
		}
		for socket.State().IsPreestablished() {
			time.Sleep(5 * time.Millisecond)
		}
		for {
			n, err := socket.Read(buf[:])
			if errors.Is(err, net.ErrClosed) {
				break
			}
			if n == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			_, err = socket.Write(buf[:n])
			if err != nil {
				return err
			}
		}
		socket.Close()
		socket.FlushOutputBuffer()
		time.Sleep(time.Second)
	}
}
