package component

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/xpt2046"
)

type Display struct {
	screen *ili9341.Device
	touch  *xpt2046.Device

	backlight OutputPin
}

// LCD Screen wiring (ILI9341)
// --------------------------
// GP10 -> SCL
// GP11 -> SDA

// GP6 -> RES (RESX)
// GP7 -> D/C (WRX)
// GP8 -> CS (CSX)

// Groud -> GND
// GP14 -> VCC
// GP15 -> BL (probably BackLight)

// FM = RDX
// --------------------------

// Touch Screen wiring (XPT2046)
// --------------------------
// GP18 -> DCLK (T_SCK)
// GP17 -> CS (T_CS)
// GP19 -> DIN (T_MOSI)
// GP16 -> DOUT (T_MISO)
// GP20 -> IRQ (T_PENIRQ)
// Groud -> GND
// --------------------------

func NewDisplay() *Display {
	NewOutputPin(machine.GP14).High() // Main Power
	backlight := NewOutputPin(machine.GP15)
	backlight.High()

	return &Display{
		screen: NewLCDScreen(
			NewSpiOutput(machine.SPI1, machine.SPI1_SCK_PIN, machine.SPI1_SDO_PIN),
			machine.GP6,
			machine.GP7,
			machine.GP8),
		touch: NewTouchScreen(
			machine.GP18,
			machine.GP17,
			machine.GP19,
			machine.GP16,
			machine.GP20),
		backlight: backlight,
	}
}

// Placeholder
func (this *Display) Draw() error {
	this.screen.Display()
	return this.screen.FillRectangle(10, 10, 100, 100, color.RGBA{R: 255, A: 255})
}
