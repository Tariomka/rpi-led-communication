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
	Logger       *slog.Logger
}

func NewRunner(config RunnerConfig) Runner {
	logger := common.NewStructuredLogger(machine.USBCDC, slog.LevelDebug)
	return &PicoRunner{
		PicoW:        controller.NewPicoW(controller.PicoConfig(config), logger),
		LayoutWorker: &led.LedLayout{},
		Logger:       logger,
	}
}

func (this *PicoRunner) Start() {
	if err := this.connectAndListen(); err != nil {
		panic(err.Error())
	}

	this.PicoW.Blink(1)
	if this.Server == nil {
		panic(common.ErrServerNotInitialized.Error())
	}

	this.PicoW.TurnLed(true)
	// go this.receiveUartMessages()
	this.PicoW.StartScreen()
	this.Server.Start()
}

func (this *PicoRunner) Stop() {
	if this.Server != nil {
		defer this.Server.Stop()
	}
}

func (this *PicoRunner) connectAndListen() error {
	if err := this.PicoW.Connect(); err != nil {
		return err
	}

	this.PicoW.Blink(1)
	listener, err := this.PicoW.GetListener()
	if err != nil {
		return err
	}

	this.PicoW.Blink(1)
	this.Server, err = tcp.NewServer(listener, this.Logger)
	return err
}

func (this *PicoRunner) receiveUartMessages() {
	for {
		this.PicoW.ReceiveFromUart()
	}
}
