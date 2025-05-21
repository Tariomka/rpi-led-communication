package controller

import (
	"log/slog"
	"machine"
	"net"

	"github.com/Tariomka/rpi-led-communication/internal/component"
)

type Board interface {
	Connect() error // Connects to Wi-Fi. Returns an error if connection process fails.
	GetListener() (net.Listener, error)
	ListenToUart()
	Blink(times uint)
	TurnLed(on bool)
}

type PicoConfig struct {
	SSID     string
	Password string
	IP       string
	Port     uint16
	Hostname string
}

type PicoW struct {
	wirelessChip *component.WirelessChip
	uart         component.UART

	logger *slog.Logger
	config PicoConfig
}

func NewPicoW(config PicoConfig, logger *slog.Logger) Board {
	return &PicoW{
		wirelessChip: component.NewWirelessChip(logger),
		uart:         component.NewConfiguredUart(machine.UART0, machine.GP3),
		logger:       logger,
		config:       config,
	}
}

func (this *PicoW) Connect() error {
	return this.wirelessChip.Connect(
		this.config.SSID,
		this.config.Password,
		this.config.IP,
		this.config.Hostname)
}

func (this *PicoW) GetListener() (net.Listener, error) {
	return this.wirelessChip.GetListener(this.config.Port)
}

func (this *PicoW) ListenToUart() {
	buffer := make([]byte, 1024)
	for {
		n, err := this.uart.Read(buffer)
		if err != nil {
			this.logger.Warn("Unexpected error while listening to UART", "error", err)
			break
		}

		if n > 0 {
			this.logger.Info("Message from STM32", "payload", string(buffer[:n]))
		}
	}

	this.logger.Info("Stopping listening to UART")
}

func (this *PicoW) Blink(times uint) {
	for range times {
		this.wirelessChip.Blink()
	}
}

func (this *PicoW) TurnLed(on bool) {
	this.wirelessChip.TurnLed(on)
}
