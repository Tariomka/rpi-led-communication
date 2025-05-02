package runner

import (
	"log/slog"
	"machine"

	"github.com/Tariomka/led-common-lib/pkg/led"
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
	LayoutWorker led.LayoutWorker
	Server       tcp.Server

	settings        RunnerConfig
	handlingPackets bool
}

func NewRunner(config RunnerConfig) Runner {
	return &PicoRunner{
		PicoW:        controller.NewPicoW(config.Hostname, config.Logger),
		LayoutWorker: &led.LedLayout{},
		settings:     config,
	}
}

func (this *PicoRunner) Start() {
	if err := this.connect(); err != nil {
		panic(err.Error())
	}

	this.PicoW.Blink(3)
	if this.Server == nil {
		panic(common.ErrServerNotInitialized.Error())
	}

	this.PicoW.TurnLed(true)
	this.Server.Start()
}

func (this *PicoRunner) Stop() {
	if this.Server != nil {
		defer this.Server.Stop()
	}
}

func (this *PicoRunner) connect() error {
	if err := this.PicoW.Connect(this.settings.SSID, this.settings.Password, this.settings.IP); err != nil {
		return err
	}

	this.PicoW.Blink(1)
	listener, err := this.PicoW.GetListener(uint16(this.settings.Port))
	if err != nil {
		return err
	}

	this.PicoW.Blink(1)
	this.Server, err = tcp.NewServer(tcp.ServerConfig{
		Listener: listener,
		Logger:   common.NewStructuredLogger(machine.USBCDC, slog.LevelDebug),
	})
	return err
}
