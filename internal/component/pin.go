package component

import "machine"

// Pin that's setup for Input mode
type InputPin struct{ machine.Pin }

func NewInputPin(pin machine.Pin) InputPin {
	input := InputPin{Pin: pin}
	input.Pin.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	return input
}

// Pin that's setup for Output mode
type OutputPin struct{ machine.Pin }

func NewOutputPin(pin machine.Pin) OutputPin {
	output := OutputPin{Pin: pin}
	output.Pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	return output
}

// func NewSpiOutput(spi machine.SPI, sck, sdo machine.Pin) machine.SPI {
// 	spi.Configure(machine.SPIConfig{
// 		Frequency: 100 * machine.KHz,
// 		SCK:       sck,
// 		SDO:       sdo,
// 		LSBFirst:  true,
// 		Mode:      1,
// 	})

// 	return spi
// }

// func NewOnBoardI2C() *machine.I2C {
// 	i2c := machine.I2C0
// 	i2c.Configure(machine.I2CConfig{
// 		Frequency: 100_000,
// 		SCL:       machine.I2C0_SCL_PIN,
// 		SDA:       machine.I2C0_SDA_PIN,
// 	})
// 	return i2c
// }
