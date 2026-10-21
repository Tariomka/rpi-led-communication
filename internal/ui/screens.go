package ui

import (
	"image/color"
	"strconv"

	"github.com/Tariomka/led-common-lib/pkg/communication/uart"
	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/Tariomka/rpi-led-communication/internal/component"
	"github.com/Tariomka/rpi-led-communication/internal/services"
)

const (
	statusRowHeight = 16
	controlColumns  = 4
	controlCount    = 12 // buttons, in rows of controlColumns
	controlWidth    = component.ScreenWidth / controlColumns
	controlHeight   = 52
	textModeHint    = "Uploaded shows play only in Text mode (SW1+SW2 ON)"
	maxCartoon      = 99 // picker limit while the cartoon count is unknown
)

var uploadStates = [...]string{"idle", "connecting", "sending", "finishing", "done", "failed"}

func (this *UI) drawStatus() {
	state := this.cube.State()

	cube, cubeColor := "responding", common.ColorGreen
	if !state.Responding {
		cube, cubeColor = "NOT RESPONDING", common.ColorRed
	}
	hint := ""
	if !isTextMode(state) {
		hint = textModeHint
	}
	playModel := "repeat one"
	if state.PlayAll {
		playModel = "all cartoons"
	}
	display := "on"
	if state.DisplayOff {
		display = "off"
	}
	started := "-"
	if state.StartedAt != 0 {
		started = strconv.Itoa(int(state.StartedAt/1000)) + " s after Pico start"
	}
	desktop := "not connected"
	if this.desktop {
		desktop = "connected"
	}

	rows := [...]struct {
		text string
		fg   color.RGBA
	}{
		{"Cube: " + cube, cubeColor},
		{"Mode: " + orUnknown(state.Mode), common.ColorWhite},
		{hint, common.ColorYellow},
		{"Cartoon: " + strconv.Itoa(state.Cartoon) + " / " + strconv.Itoa(state.Count), common.ColorWhite},
		{"Speed: " + speed(state.Speed), common.ColorWhite},
		{"Play model: " + playModel, common.ColorWhite},
		{"Paused: " + yesNo(state.Pause), common.ColorWhite},
		{"Display: " + display, common.ColorWhite},
		{"Cube type: " + orUnknown(state.Type), common.ColorWhite},
		{"Device number: " + strconv.Itoa(state.DeviceNum), common.ColorWhite},
		{"Cube restarted: " + started, common.ColorWhite},
		{"Wi-Fi: " + this.network, common.ColorWhite},
		{"Desktop: " + desktop, common.ColorWhite},
	}
	for i, row := range rows {
		this.row(int16(i)*statusRowHeight+2, row.text, row.fg)
	}
}

func (this *UI) controlLabels(state services.CubeState) (labels [controlCount]string, lit [controlCount]bool) {
	playModel := "6 Repeat 1"
	if state.PlayAll {
		playModel = "6 Play all"
	}
	labels = [controlCount]string{
		"1 Previous", "2 Next", "3 Faster", "4 Slower",
		"5 Pause", playModel, "7 Disp. off", "Replay",
		"Speed reset", "-", "Play " + strconv.Itoa(this.picker), "+",
	}
	lit[4], lit[5], lit[6] = state.Pause, state.PlayAll, state.DisplayOff
	return labels, lit
}

func (this *UI) drawControl() {
	state := this.cube.State()
	labels, lit := this.controlLabels(state)
	for i, label := range labels {
		fg, bg := common.ColorWhite, common.ColorNavy
		if lit[i] {
			fg, bg = common.ColorBlack, common.ColorOrange
		}
		x, y := int16(i%controlColumns)*controlWidth, int16(i/controlColumns)*controlHeight
		this.button(x, y, controlWidth, controlHeight, label, fg, bg)
	}

	y := int16(controlCount/controlColumns)*controlHeight + 2
	this.row(y, "Cartoon "+strconv.Itoa(state.Cartoon)+" / "+strconv.Itoa(state.Count)+
		", speed "+speed(state.Speed), common.ColorWhite)
	hint := ""
	if !isTextMode(state) {
		hint = textModeHint
	}
	this.row(y+component.TextHeight, hint, common.ColorYellow)
	cube := ""
	if !state.Responding {
		cube = "Cube not responding"
	}
	this.row(y+2*component.TextHeight, cube, common.ColorRed)
}

func (this *UI) tapControl(x, y int16) {
	i := int(y/controlHeight)*controlColumns + int(x/controlWidth)
	if i >= controlCount {
		return
	}

	var err error
	switch i {
	case 0:
		err = this.cube.Key(services.KeyPreviousCartoon)
	case 1:
		err = this.cube.Key(services.KeyNextCartoon)
	case 2:
		err = this.cube.Key(services.KeySpeedUp)
	case 3:
		err = this.cube.Key(services.KeySpeedDown)
	case 4:
		err = this.cube.Key(services.KeyPause)
	case 5:
		err = this.cube.Key(services.KeyPlayModel)
	case 6:
		err = this.cube.Key(services.KeyDisplayOff)
	case 7:
		err = this.cube.Action(services.ActionReplay)
	case 8:
		err = this.cube.Action(services.ActionSpeedReset)
	case 9:
		this.picker = max(0, this.picker-1)
	case 10:
		err = this.cube.Cartoon(this.picker)
	case 11:
		last := maxCartoon
		if count := this.cube.State().Count; count > 0 {
			last = count - 1
		}
		this.picker = min(last, this.picker+1)
	}

	this.redrawContent = true
	if err != nil {
		this.logger.Warn("Cube command failed", "error", err)
	}
}

func (this *UI) drawLog() {
	this.logTotal = this.log.Total()
	n := this.log.Last(this.lines[:])
	for row := range logRows {
		text, fg := "", common.ColorWhite
		if i := row - (logRows - n); i >= 0 {
			text = this.lines[i].Text
			if this.lines[i].Source == services.SourcePico {
				fg = common.ColorCyan
			}
		}
		this.row(int16(row)*component.TextHeight, text, fg)
	}
}

func (this *UI) drawUpload() {
	const barX, barY, barWidth, barHeight = 8, 56, component.ScreenWidth - 16, 16

	uploadState, acked, total, result := this.cube.Upload()
	this.row(2, "Show upload", common.ColorCyan)
	this.row(20, "State: "+uploadStates[uploadState], common.ColorWhite)
	this.row(36, "Stored: "+strconv.Itoa(acked)+" / "+strconv.Itoa(total)+" bytes", common.ColorWhite)

	filled := int16(0)
	if total > 0 {
		filled = int16(acked * barWidth / total)
	}
	this.display.Fill(barX, barY, filled, barHeight, common.ColorGreen)
	this.display.Fill(barX+filled, barY, barWidth-filled, barHeight, common.ColorDarkGray)

	resultColor := common.ColorWhite
	if uploadState == uart.UploadFailed {
		resultColor = common.ColorRed
	}
	this.row(84, "Last result: "+orUnknown(result), resultColor)
	this.row(116, textModeHint, common.ColorYellow)
	this.row(132, "Cube mode: "+orUnknown(this.cube.State().Mode), common.ColorWhite)
}

func isTextMode(state services.CubeState) bool {
	return state.Mode == "" || state.Mode == "Text"
}

// speed formats a speed in 1/100.
func speed(value int) string {
	fraction := strconv.Itoa(value % 100)
	if len(fraction) < 2 {
		fraction = "0" + fraction
	}
	return strconv.Itoa(value/100) + "." + fraction
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func orUnknown(text string) string {
	if text == "" {
		return "-"
	}
	return text
}
