package component

import (
	"machine"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"tinygo.org/x/drivers/touch"
	"tinygo.org/x/drivers/xpt2046"
	"tinygo.org/x/tinyfont/proggy"
	"tinygo.org/x/tinyterm"
)

type Display struct {
	screen *tinyterm.Terminal
	touch  *xpt2046.Device

	backlight OutputPin
}

// LCD Screen wiring (ILI9341)
// --------------------------
// GP10 -> SCL
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

	println("Initializing SPI...")
	spi := NewSpiOutput(machine.SPI1, machine.SPI1_SCK_PIN, machine.SPI1_SDO_PIN)
	println("Initializing display...")
	screen := NewLCDScreen(
		spi,
		machine.GP6,
		machine.GP7,
		machine.GP8)
	println("Covering in red...")
	screen.FillScreen(common.ColorRed)

	terminal := tinyterm.NewTerminal(screen)
	terminal.Configure(&tinyterm.Config{
		Font:              &proggy.TinySZ8pt7b,
		FontHeight:        8,
		FontOffset:        6,
		UseSoftwareScroll: true,
	})

	return &Display{
		screen: terminal,
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
func (this *Display) Draw() {
	this.screen.Println("Hello from TinyTerm!")
	// return this.screen.FillRectangle(10, 10, 100, 100, color.RGBA{R: 255, A: 255})
	this.screen.Display()
}

func (this *Display) ReadTouch() touch.Point {
	if this.touch.Touched() {
		return this.touch.ReadTouchPoint()
	}

	return touch.Point{}
}
