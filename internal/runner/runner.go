package runner

import (
	"fmt"
	"log/slog"
	"machine"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/Tariomka/rpi-led-communication/internal/controller"
	"github.com/Tariomka/rpi-led-communication/internal/tcp"
)

type Runner interface {
	Start()
	Stop()
}

type PicoRunner struct {
	PicoW        controller.Board
	LayoutWorker controller.LayoutWorker
	Server       tcp.Server

	settings        RunnerConfig
	handlingPackets bool
}

func NewRunner(config RunnerConfig) (Runner, error) {
	fmt.Printf("Data: %s\n", config)

	// server, err := tcp.NewServer(tcp.ServerConfig{Address: config.ListenAddress})
	// if err != nil {
	// 	return nil, err
	// }

	runner := &PicoRunner{
		PicoW:        controller.NewPicoW(config.Hostname, config.Logger),
		LayoutWorker: &controller.LedLayout{},
		// Server:       server,
		settings: config,
	}

	return runner, nil
}

func (pr *PicoRunner) Start() {
	if err := pr.connect(); err != nil {
		// if pr.connect() != nil {
		panic(err.Error())
	}
	if pr.Server == nil {
		panic("server not initialized")
	}
	pr.Server.Start()
	// testOut()
}

func (pr *PicoRunner) Stop() {
	if pr.Server != nil {
		defer pr.Server.Stop()
	}
}

func (pr *PicoRunner) connect() error {
	if err := pr.PicoW.Connect(pr.settings.SSID, pr.settings.Password, "192.168.0.169"); err != nil {
		return err
	}

	listener, err := pr.PicoW.GetListener(42069)
	if err != nil {
		return err
	}

	pr.Server, err = tcp.NewServer(tcp.ServerConfig{
		Listener: listener,
		Logger: slog.New(common.NewLogHandler(
			func(message string) { machine.USBCDC.Write([]byte(message + "\n")) },
			&slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	return err
}
