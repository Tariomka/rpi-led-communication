package controller

import (
	"log/slog"
	"machine"
	"net/netip"

	"github.com/Tariomka/rpi-led-communication/internal/component"
)

type PicoConfig struct {
	SSID     string
	Password string
	IP       string
	Port     uint16
	Hostname string
}

// PicoW owns the board's hardware: the touchscreen, the cube UART and the Wi-Fi chip.
type PicoW struct {
	Display *component.Display
	Cube    component.UART // UART0: GP0 TX -> cube RXD, GP1 RX <- cube TXD

	wirelessChip *component.WirelessChip
	config       PicoConfig
}

func NewPicoW(config PicoConfig, logger *slog.Logger) *PicoW {
	cube := component.NewConfiguredUart(machine.UART0)
	return &PicoW{
		Display:      component.NewDisplay(), // Must be registered before Cyw43439
		Cube:         cube,
		wirelessChip: component.NewWirelessChip(logger),
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

func (this *PicoW) GetListener() (*component.Listener, error) {
	return this.wirelessChip.GetListener(this.config.Port)
}

func (this *PicoW) Addr() netip.Addr {
	return this.wirelessChip.Addr()
}

func (this *PicoW) Blink(times uint) {
	for range times {
		this.wirelessChip.Blink()
	}
}

func (this *PicoW) TurnLed(on bool) {
	this.wirelessChip.TurnLed(on)
}
