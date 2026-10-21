package ui

import (
	"image/color"
	"log/slog"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/Tariomka/rpi-led-communication/internal/component"
	"github.com/Tariomka/rpi-led-communication/internal/services"
)

type Tab int

const (
	TabStatus Tab = iota
	TabControl
	TabLog
	TabUpload
	tabCount
)

const (
	contentHeight  = 208
	tabWidth       = component.ScreenWidth / int16(tabCount)
	tabHeight      = component.ScreenHeight - contentHeight
	redrawInterval = 100 // ms between content redraws
	logRows        = contentHeight / component.TextHeight
)

var tabNames = [tabCount]string{"Status", "Control", "Log", "Upload"}

// UI is the touchscreen: a tab bar at the bottom and one screen per tab. Run Update from the main
// loop.
type UI struct {
	display     *component.Display
	cube        *services.CubeService
	log         *services.LineLog
	logger      *slog.Logger
	touch       touchInput
	calibration Calibration

	tab           Tab
	redrawAll     bool
	redrawContent bool
	drawnAt       uint32

	network   string
	desktop   bool
	uploading bool
	picker    int // cartoon index for the Play button
	logPaused bool
	logTotal  uint32
	lines     [logRows]services.LogLine
}

func New(
	display *component.Display,
	cube *services.CubeService,
	log *services.LineLog,
	calibration Calibration,
	logger *slog.Logger) *UI {
	return &UI{
		display:     display,
		cube:        cube,
		log:         log,
		logger:      logger,
		touch:       touchInput{display: display},
		calibration: calibration,
		redrawAll:   true,
		network:     "connecting...",
	}
}

func (this *UI) SetNetwork(text string) {
	this.network, this.redrawContent = text, true
}

func (this *UI) SetDesktop(connected bool) {
	this.desktop, this.redrawContent = connected, true
}

// Update handles touch input and redraws what changed; now is Millis.
func (this *UI) Update(now uint32) {
	if rawX, rawY, ok := this.touch.poll(now); ok && this.calibration.Valid() {
		this.tap(this.calibration.point(rawX, rawY))
	}

	if this.cube.Changed() {
		this.redrawContent = true
	}
	if uploading := this.cube.Uploading(); uploading != this.uploading {
		this.uploading = uploading
		if uploading {
			this.selectTab(TabUpload)
		}
	}
	if this.tab == TabLog && !this.logPaused && this.log.Total() != this.logTotal {
		this.redrawContent = true
	}

	switch {
	case this.redrawAll:
		this.redrawAll, this.redrawContent = false, false
		this.drawTabs()
		this.display.Fill(0, 0, component.ScreenWidth, contentHeight, common.ColorBlack)
		this.drawContent()
	case this.redrawContent && now-this.drawnAt >= redrawInterval:
		this.redrawContent = false
		this.drawContent()
	default:
		return
	}
	this.drawnAt = now
}

func (this *UI) selectTab(tab Tab) {
	if tab != this.tab {
		this.tab, this.redrawAll = tab, true
	}
}

func (this *UI) tap(x, y int16) {
	if y >= contentHeight {
		this.selectTab(Tab(x / tabWidth))
		return
	}

	switch this.tab {
	case TabControl:
		this.tapControl(x, y)
	case TabLog:
		this.logPaused, this.redrawAll = !this.logPaused, true
	}
}

func (this *UI) drawTabs() {
	for tab := range tabCount {
		label := tabNames[tab]
		if tab == TabLog && this.logPaused {
			label = "Log paused"
		}
		background := common.ColorDarkGray
		if tab == this.tab {
			background = common.ColorDarkCyan
		}
		this.button(int16(tab)*tabWidth, contentHeight, tabWidth, tabHeight, label, common.ColorWhite, background)
	}
}

func (this *UI) drawContent() {
	switch this.tab {
	case TabStatus:
		this.drawStatus()
	case TabControl:
		this.drawControl()
	case TabLog:
		this.drawLog()
	case TabUpload:
		this.drawUpload()
	}
}

// button draws a labeled button inside the cell, leaving a 2 px gap around it.
func (this *UI) button(x, y, w, h int16, label string, fg, bg color.RGBA) {
	x, y, w, h = x+2, y+2, w-4, h-4
	top := (h - component.TextHeight) / 2
	this.display.Fill(x, y, w, top, bg)
	this.display.Text(x, y+top, w, label, fg, bg, true)
	this.display.Fill(x, y+top+component.TextHeight, w, h-top-component.TextHeight, bg)
}

// row draws one full width text line.
func (this *UI) row(y int16, text string, fg color.RGBA) {
	this.display.Text(0, y, component.ScreenWidth, text, fg, common.ColorBlack, false)
}
