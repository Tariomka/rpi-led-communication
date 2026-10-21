package ui

import (
	"image/color"
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/Tariomka/rpi-led-communication/internal/component"
)

const (
	touchInterval = 25 // ms between touch polls
	minPress      = 30 // ms, shorter presses are noise
	crossSize     = 8
)

var calibrationTargets = [3][2]int16{{20, 20}, {300, 20}, {20, 220}}

// Calibration maps raw XPT2046 readings to screen pixels.
type Calibration struct {
	SwapXY bool // screen x comes from the raw y channel
	X0, X1 int  // raw readings at screen x = 0 and x = ScreenWidth-1
	Y0, Y1 int  // raw readings at screen y = 0 and y = ScreenHeight-1
}

func (this Calibration) Valid() bool {
	return this.X0 != this.X1 && this.Y0 != this.Y1
}

func (this Calibration) point(rawX, rawY int) (x, y int16) {
	if this.SwapXY {
		rawX, rawY = rawY, rawX
	}
	return scale(rawX, this.X0, this.X1, component.ScreenWidth), scale(rawY, this.Y0, this.Y1, component.ScreenHeight)
}

func scale(raw, atStart, atEnd, size int) int16 {
	return int16(max(0, min(size-1, (raw-atStart)*(size-1)/(atEnd-atStart))))
}

// edges extrapolates the raw readings at two screen positions to the first and last pixel.
func edges(raw0, raw1, at0, at1, size int) (start, end int) {
	return raw0 - (raw1-raw0)*at0/(at1-at0), raw0 + (raw1-raw0)*(size-1-at0)/(at1-at0)
}

type touchInput struct {
	display   *component.Display
	pollAt    uint32
	pressed   bool
	pressedAt uint32
	sumX      int
	sumY      int
	samples   int
}

// poll returns the averaged raw position of a press once it is released.
func (this *touchInput) poll(now uint32) (rawX, rawY int, ok bool) {
	if int32(now-this.pollAt) < 0 {
		return 0, 0, false
	}
	this.pollAt = now + touchInterval

	if this.display.Touched() {
		if !this.pressed {
			this.pressed, this.pressedAt = true, now
			this.sumX, this.sumY, this.samples = 0, 0, 0
		}
		if x, y, ok := this.display.ReadTouch(); ok {
			this.sumX, this.sumY, this.samples = this.sumX+x, this.sumY+y, this.samples+1
		}
		return 0, 0, false
	}

	if !this.pressed {
		return 0, 0, false
	}
	this.pressed = false
	if now-this.pressedAt < minPress || this.samples == 0 {
		return 0, 0, false
	}
	return this.sumX / this.samples, this.sumY / this.samples, true
}

// Calibrate asks for taps on three crosses until it gets a usable calibration, and uses it.
func (this *UI) Calibrate() Calibration {
	for {
		this.display.Fill(0, 0, component.ScreenWidth, component.ScreenHeight, common.ColorBlack)
		this.display.Text(0, 110, component.ScreenWidth, "Touch calibration: tap each cross", common.ColorWhite, common.ColorBlack, true)

		var raw [len(calibrationTargets)][2]int
		for i, target := range calibrationTargets {
			this.drawCross(target, common.ColorYellow)
			raw[i] = this.waitTap()
			this.drawCross(target, common.ColorBlack)
		}

		// From the first to the second target only the screen x changes: the raw channel that
		// moved the most is the x channel.
		var calibration Calibration
		calibration.SwapXY = abs(raw[1][1]-raw[0][1]) > abs(raw[1][0]-raw[0][0])
		if calibration.SwapXY {
			for i := range raw {
				raw[i][0], raw[i][1] = raw[i][1], raw[i][0]
			}
		}

		calibration.X0, calibration.X1 = edges(raw[0][0], raw[1][0],
			int(calibrationTargets[0][0]), int(calibrationTargets[1][0]), component.ScreenWidth)
		calibration.Y0, calibration.Y1 = edges(raw[0][1], raw[2][1],
			int(calibrationTargets[0][1]), int(calibrationTargets[2][1]), component.ScreenHeight)
		if calibration.Valid() {
			this.calibration, this.redrawAll = calibration, true
			return calibration
		}
	}
}

func (this *UI) waitTap() [2]int {
	for {
		if x, y, ok := this.touch.poll(common.Millis()); ok {
			return [2]int{x, y}
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (this *UI) drawCross(target [2]int16, c color.RGBA) {
	x, y := target[0], target[1]
	this.display.Fill(x-crossSize, y, 2*crossSize+1, 1, c)
	this.display.Fill(x, y-crossSize, 1, 2*crossSize+1, c)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
