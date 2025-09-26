package component

import (
	"fmt"
	"machine"
	"strings"
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/touch"
	"tinygo.org/x/drivers/xpt2046"
	"tinygo.org/x/tinyfont/proggy"
	"tinygo.org/x/tinyterm"
)

type Display struct {
	lcd    *ili9341.Device
	screen *tinyterm.Terminal
	touch  *xpt2046.Device

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

	terminal := tinyterm.NewTerminal(screen)
	terminal.Configure(&tinyterm.Config{
		Font:              &proggy.TinySZ8pt7b,
		FontHeight:        14,
		FontOffset:        6,
		UseSoftwareScroll: false,
	})

	return &Display{
		lcd:    screen,
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
	this.lcd.FillScreen(common.ColorBlack)

	this.screen.Println("Hello from TinyTerm!")
	this.screen.Display()
}

func (this *Display) Draw2() {
	for n := 0; ; n++ {
		this.screen.Write([]byte("   " + strings.Repeat("_", 36) + "\n"))
		fmt.Fprintf(this.screen, "%02x|", 0)
		for n := 0; n < 16; n++ {
			fmt.Fprintf(this.screen, "\x1b[48;5;%dm \x1b[0m", n)
		}
		this.screen.Write([]byte(strings.Repeat(" ", 20) + "|\n"))
		for n := 0; n < 6; n++ {
			i := n*36 + 16
			fmt.Fprintf(this.screen, "%02x|", i)
			for j := 0; j < 36; j++ {
				v := i + j
				fmt.Fprintf(this.screen, "\x1b[48;5;%dm \x1b[0m", v)
			}
			this.screen.WriteByte('|')
			this.screen.WriteByte('\n')
		}
		fmt.Fprintf(this.screen, "%02x|", 232)
		for n := 232; n <= 255; n++ {
			fmt.Fprintf(this.screen, "\x1b[48;5;%dm \x1b[0m", n)
		}
		this.screen.Write([]byte(strings.Repeat(" ", 12) + "|\n"))
		this.screen.Write([]byte("   " + strings.Repeat("\xaf", 36) + "\n"))

		this.screen.Display()
		time.Sleep(5 * time.Second)
	}
}

func (this *Display) ReadTouch() touch.Point {
	if this.touch.Touched() {
		return this.touch.ReadTouchPoint()
	}

	return touch.Point{}
}
