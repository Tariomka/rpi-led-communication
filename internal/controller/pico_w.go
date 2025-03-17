package controller

import "github.com/Tariomka/rpi-led-communication/internal/component"

type Board interface {
}

type PicoW struct {
	WirelessChip component.Wireless // (CYW43439) WiFi + Bluetooth
}

func NewPicoW() Board {

	return &PicoW{
		WirelessChip: component.NewWirelessChip(),
	}
}
