package component

import (
	"machine"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/touch"
	"tinygo.org/x/drivers/xpt2046"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

const (
	verticalOffset   = 0
	horizontalOffset = 0
	// horizontalOffset = 42
)

type Display struct {
	lcd   *ili9341.Device
	touch *xpt2046.Device

	backlight OutputPin
}

// LCD Screen wiring (ILI9341)
// --------------------------
// GP10 -> SCL (D/CX)
// GP11 -> SDA
// GP12 -> SDO (Unused)

// GP6 -> RES (RESX)
// GP7 -> D/C (WRX)
// GP8 -> CS (CSX)
// GP9 -> FM = RDX (Unused)

// Groud -> GND
// GP14 -> VCC
// GP15 -> BL
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

	spi := NewSpiOutput(machine.SPI1, machine.SPI1_SCK_PIN, machine.SPI1_SDO_PIN)
	screen := NewLCDScreen(
		spi,
		machine.GP6,
		machine.GP7,
		machine.GP8)
	screen.FillScreen(common.ColorRed)

	return &Display{
		lcd: screen,
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
func (this *Display) DrawBackground() {
	this.lcd.FillScreen(common.ColorBlack)
}

func (this *Display) DrawText() {
	text := "Kas Skatys, Tas Gaidys!"

	// tinyfont.WriteLine(this.lcd, &proggy.TinySZ8pt7b, horizontalOffset, 70, text, common.ColorGreen)
	tinyfont.WriteLineRotated(
		this.lcd,
		&proggy.TinySZ8pt7b,
		horizontalOffset+10,
		50,
		text,
		common.ColorMagenta,
		tinyfont.ROTATION_90)

	tinyfont.WriteLineRotated(
		this.lcd,
		&proggy.TinySZ8pt7b,
		horizontalOffset+310,
		190,
		text,
		common.ColorRed,
		tinyfont.ROTATION_270)
}

func (this *Display) DrawImage() {
	image := pixel.NewImageFromBytes[pixel.RGB565BE](240, 240, embededImage)
	this.lcd.DrawBitmap(horizontalOffset+40, verticalOffset, image)
}

func (this *Display) ReadTouch() touch.Point {
	if this.touch.Touched() {
		return this.touch.ReadTouchPoint()
	}

	return touch.Point{}
}
