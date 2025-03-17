package runner

import (
	"errors"
	"log/slog"
	"machine"
	"net"
	"net/netip"
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/soypat/seqs"
	"github.com/soypat/seqs/stacks"
)

func testOut() {
	// For testing
	time.Sleep(time.Second)
	config := NewConfig()
	ssid = config.SSID
	pass = config.Password
	logger := slog.New(common.NewLogHandler(
		func(message string) { machine.Serial.Write([]byte(message + "\n")) },
		&slog.HandlerOptions{Level: slog.LevelInfo}))
	// &slog.HandlerOptions{Level: slog.LevelDebug - 2}))
	_, stack, dev, err := SetupWithDHCP(SetupConfig{
		Hostname: "TCP-pico",
		Logger:   logger,
		TCPPorts: 1,
	})

	_ = dev
	// for i := 0; i < 5; i++ {
	// 	err = dev.GPIOSet(0, true)
	// 	if err != nil {
	// 		println("err", err.Error())
	// 	} else {
	// 		println("LED ON")
	// 	}
	// 	time.Sleep(500 * time.Millisecond)
	// 	err = dev.GPIOSet(0, false)
	// 	if err != nil {
	// 		println("err", err.Error())
	// 	} else {
	// 		println("LED OFF")
	// 	}
	// 	time.Sleep(500 * time.Millisecond)
	// }

	if err != nil {
		panic("in dhcp setup:" + err.Error())
	}
	// Start TCP server.
	const socketBuf = 1024
	// const socketBuf = 256
	const listenPort = 42069
	listenAddr := netip.AddrPortFrom(stack.Addr(), listenPort)
	socket, err := stacks.NewTCPConn(stack, stacks.TCPConnConfig{TxBufSize: socketBuf, RxBufSize: socketBuf})
	if err != nil {
		panic("socket create:" + err.Error())
	}
	println("start listening on:", listenAddr.String())
	err = ForeverTCPListenEcho(socket, listenAddr)
	if err != nil {
		panic("socket listen:" + err.Error())
	}
}

func ForeverTCPListenEcho(socket *stacks.TCPConn, addr netip.AddrPort) error {
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
