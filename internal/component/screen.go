package component

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/xpt2046"
)

type Screen interface {
	Draw()
}

type ILI9341 struct {
	screen *ili9341.Device
	touch  xpt2046.Device

	power     OutputPin
	backlight OutputPin
}

// GP10 -> SCK
// GP11 -> SDI
// GP12 -> SD0

// GP6 -> RES (RESX)
// GP7 -> D/C (WRX)
// GP8 -> CS (CSX)

// Groud -> GND
// GP14 -> VCC
// GP15 -> BL (probably BackLight)

func NewScreen() Screen {
	power := NewOutputPin(machine.GP14)
	power.High()
	backlight := NewOutputPin(machine.GP15)
	backlight.High()

	spi := NewSpiOutput(machine.SPI1, machine.SPI1_SCK_PIN, machine.SPI1_SDO_PIN)
	screen := NewLCDScreen(spi, machine.GP6, machine.GP7, machine.GP8)
	touch := xpt2046.Device{}

	return &ILI9341{
		screen:    screen,
		touch:     touch,
		power:     power,
		backlight: backlight,
	}
}

func (this *ILI9341) Draw() {
	println("Drawing on screen...")
	if err := this.screen.FillRectangle(10, 10, 100, 100, color.RGBA{R: 255, A: 255}); err != nil {
		println("Error filling rectangle: ", err.Error())
	}
}
