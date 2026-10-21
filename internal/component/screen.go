package component

import (
	"image/color"
	"machine"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/xpt2046"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

const (
	ScreenWidth  = 320
	ScreenHeight = 240
	TextHeight   = 12 // pixel rows of one text line
	CharWidth    = 6  // TinySZ8pt7b is monospaced
	textBaseline = 9
	textMargin   = 4
)

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

// Display draws through a small strip buffer: a full frame buffer doesn't fit next to the Wi-Fi
// stack. The yield callback runs between strips so the UART keeps being read while drawing.
type Display struct {
	lcd   *ili9341.Device
	touch *xpt2046.Device

	backlight OutputPin
	strip     strip
	yield     func()
}

func NewDisplay() *Display {
	NewOutputPin(machine.GP14).High() // Main Power
	backlight := NewOutputPin(machine.GP15)
	backlight.High()

	spi := NewSpiOutput(machine.SPI1, machine.SPI1_SCK_PIN, machine.SPI1_SDO_PIN)
	display := &Display{
		lcd: NewLCDScreen(spi, machine.GP6, machine.GP7, machine.GP8),
		touch: NewTouchScreen(
			machine.GP18,
			machine.GP17,
			machine.GP19,
			machine.GP16,
			machine.GP20),
		backlight: backlight,
		yield:     func() {},
	}
	display.Fill(0, 0, ScreenWidth, ScreenHeight, common.ColorBlack)
	return display
}

func (this *Display) SetYield(yield func()) {
	this.yield = yield
}

func (this *Display) Fill(x, y, w, h int16, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}

	rows := min(h, int16(len(this.strip.buffer)/2/int(w)))
	this.strip.fill(w, rows, c)
	for ; h > 0; h -= rows {
		rows = min(rows, h)
		this.blit(x, y, w, rows)
		y += rows
	}
}

// Text draws one line of text on a w px wide, TextHeight px high background, cut to fit.
func (this *Display) Text(x, y, w int16, text string, fg, bg color.RGBA, center bool) {
	text = text[:min(len(text), max(0, int((w-2*textMargin)/CharWidth)))]
	left := int16(textMargin)
	if center {
		left = (w - int16(len(text))*CharWidth) / 2
	}

	this.strip.fill(w, TextHeight, bg)
	tinyfont.WriteLine(&this.strip, &proggy.TinySZ8pt7b, left, textBaseline, text, fg)
	this.blit(x, y, w, TextHeight)
}

func (this *Display) Touched() bool {
	return this.touch.Touched()
}

// ReadTouch returns the raw touch position; ok is false when the press ended during the read.
func (this *Display) ReadTouch() (x, y int, ok bool) {
	point := this.touch.ReadTouchPoint()
	return point.X, point.Y, point.X != 0 || point.Y != 0
}

func (this *Display) blit(x, y, w, h int16) {
	this.lcd.DrawRGBBitmap8(x, y, this.strip.buffer[:int(w)*int(h)*2], w, h)
	this.yield()
}

// strip is a RGB565 (big endian) pixel buffer for up to ScreenWidth x TextHeight pixels.
type strip struct {
	buffer [ScreenWidth * TextHeight * 2]byte
	width  int16
	height int16
}

func (this *strip) fill(w, h int16, c color.RGBA) {
	this.width, this.height = w, h
	value := ili9341.RGBATo565(c)
	high, low := byte(value>>8), byte(value)
	for i := 0; i < int(w)*int(h)*2; i += 2 {
		this.buffer[i], this.buffer[i+1] = high, low
	}
}

func (this *strip) Size() (x, y int16) { return this.width, this.height }

func (this *strip) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= this.width || y >= this.height {
		return
	}

	value := ili9341.RGBATo565(c)
	i := (int(y)*int(this.width) + int(x)) * 2
	this.buffer[i], this.buffer[i+1] = byte(value>>8), byte(value)
}

func (this *strip) Display() error { return nil }
