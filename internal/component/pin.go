package component

import (
	"machine"

	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/xpt2046"
)

// Pin that's setup for Input mode
type InputPin struct{ machine.Pin }

func NewInputPin(pin machine.Pin) InputPin {
	input := InputPin{Pin: pin}
	input.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	return input
}

// Pin that's setup for Output mode
type OutputPin struct{ machine.Pin }

func NewOutputPin(pin machine.Pin) OutputPin {
	output := OutputPin{Pin: pin}
	output.Configure(machine.PinConfig{Mode: machine.PinOutput})
	return output
}

func NewSpiOutput(spi *machine.SPI, sck, sdo machine.Pin) *machine.SPI {
	spi.Configure(machine.SPIConfig{
		Frequency: 40 * machine.MHz,
		SCK:       sck,
		SDO:       sdo,
		Mode:      3,
	})

	return spi
}

type UART struct{ *machine.UART }

func NewConfiguredUart(uart *machine.UART, powerPin machine.Pin) UART {
	device := UART{UART: uart}
	device.Configure(machine.UARTConfig{})

	NewOutputPin(powerPin).High()

	return device
}

func NewLCDScreen(spi *machine.SPI, reset, dc, cs machine.Pin) *ili9341.Device {
	device := ili9341.NewSPI(spi, dc, cs, reset)
	device.Configure(ili9341.Config{})

	return device
}

func NewTouchScreen() xpt2046.Device {
	touch := xpt2046.Device{}

	return touch
}
