package runner

import (
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

func NewRunner(config RunnerConfig) Runner {
	return &PicoRunner{
		PicoW:        controller.NewPicoW(config.Hostname, config.Logger),
		LayoutWorker: &controller.LedLayout{},
		settings:     config,
	}
}

func (pr *PicoRunner) Start() {
	if err := pr.connect(); err != nil {
		panic(err.Error())
	}

	if pr.Server == nil {
		panic(common.ErrServerNotInitialized.Error())
	}

	pr.Server.Start()
}

func (pr *PicoRunner) Stop() {
	if pr.Server != nil {
		defer pr.Server.Stop()
	}
}

func (pr *PicoRunner) connect() error {
	if err := pr.PicoW.Connect(pr.settings.SSID, pr.settings.Password, pr.settings.IP); err != nil {
		return err
	}

	listener, err := pr.PicoW.GetListener(uint16(pr.settings.Port))
	if err != nil {
		return err
	}

	pr.Server, err = tcp.NewServer(tcp.ServerConfig{
		Listener: listener,
		Logger:   common.NewStructuredLogger(machine.USBCDC, slog.LevelDebug),
	})
	return err
}
