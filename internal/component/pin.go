package component

import "machine"

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

type UART struct {
	*machine.UART
	power OutputPin
}

func NewConfiguredUart(uart *machine.UART, powerPin machine.Pin) UART {
	device := UART{
		UART:  uart,
		power: NewOutputPin(powerPin),
	}
	device.Configure(machine.UARTConfig{})
	device.power.Pin.High()
	return device
}
