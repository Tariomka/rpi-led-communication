package component

import (
	"device/rp"
	"fmt"
	"machine"
	"strconv"
	"unsafe"
)

// WatchdogTimeout is how long the main loop may stall before the Pico restarts.
const WatchdogTimeout = 5000 // ms

// Brown-out threshold on the 1.1 V core supply; the default 0.860 V only trips on a deep drop.
const brownOutLevel = 0b1011 // 0.946 V

// Stage is where the main loop is; it is recorded so a watchdog timeout can be located.
type Stage uint32

const (
	StageStartup Stage = iota + 1
	StageLink
	StageCube
	StageDesktop
	StageScreen
)

var stageNames = [...]string{"", "startup", "Wi-Fi link", "cube poll", "desktop request", "screen update"}

func (this Stage) String() string {
	if int(this) < len(stageNames) {
		return stageNames[this]
	}
	return "stage " + strconv.Itoa(int(this))
}

type ResetCause int

const (
	ResetPowerOn  ResetCause = iota // power applied, or a brown-out when the last run is known
	ResetButton                     // RUN pin
	ResetDebugger                   // PSM restart
	ResetWatchdog                   // the main loop stalled in Stage
	ResetCrash                      // a goroutine panicked, see Panic
	ResetForced                     // bootrom, e.g. after flashing
)

var causeNames = [...]string{"power-on", "reset button", "debugger", "watchdog timeout", "crash", "forced reset"}

func (this ResetCause) String() string { return causeNames[this] }

// ResetInfo is why the Pico restarted and what the last run was doing, when known.
type ResetInfo struct {
	Cause    ResetCause
	LastRun  bool   // Stage and Uptime are valid: RAM survived the reset
	Stage    Stage  // main loop stage of the last run
	Uptime   uint32 // ms the last run lasted
	Panic    string // "where: message" for ResetCrash
	BrownOut bool   // ResetPowerOn while the last run is known: the supply dropped, not a cold start
}

func (this ResetInfo) String() string {
	text := this.Cause.String()
	switch {
	case this.Cause == ResetCrash:
		text += ": " + this.Panic
	case this.Cause == ResetWatchdog:
		text += " in " + this.Stage.String()
	case this.BrownOut:
		text = "brown-out (power dropped while running)"
	}
	if this.LastRun {
		text += ", last run " + strconv.Itoa(int(this.Uptime/1000)) + " s"
	}
	return text
}

// lastRun is kept in SRAM4 (0x20040000, 4 KB), which TinyGo's linker script and the bootrom leave
// alone and which keeps its content through a watchdog reset. The watchdog scratch registers are
// too small for a panic message. A power-on leaves random content, so it is checked with a magic.
type lastRun struct {
	magic  uint32
	stage  uint32
	uptime uint32
	kind   uint32 // 0 running, 1 panic stored
	length uint32
	text   [240]byte
}

const (
	lastRunAddress = 0x20040000
	lastRunMagic   = 0x1ed5cafe
	kindRunning    = 0
	kindPanic      = 1
)

var record = (*lastRun)(unsafe.Pointer(uintptr(lastRunAddress)))

// ReadReset returns why the Pico started, clears the stored reasons and arms the brown-out detector
// and the watchdog reset. Call it first thing at boot, then StartWatchdog once the start-up work
// that may block is done.
func ReadReset() ResetInfo {
	info := ResetInfo{}
	if record.magic == lastRunMagic {
		info.LastRun = true
		info.Stage, info.Uptime = Stage(record.stage), record.uptime
	}

	switch {
	case rp.WATCHDOG.GetREASON_TIMER() != 0:
		info.Cause = ResetWatchdog
	case rp.WATCHDOG.GetREASON_FORCE() != 0 && info.LastRun && record.kind == kindPanic:
		info.Cause = ResetCrash
		info.Panic = string(record.text[:min(record.length, uint32(len(record.text)))])
	case rp.WATCHDOG.GetREASON_FORCE() != 0:
		info.Cause = ResetForced
	case rp.VREG_AND_CHIP_RESET.GetCHIP_RESET_HAD_PSM_RESTART() != 0:
		info.Cause = ResetDebugger
	case rp.VREG_AND_CHIP_RESET.GetCHIP_RESET_HAD_RUN() != 0:
		info.Cause = ResetButton
	default:
		info.Cause = ResetPowerOn
		info.BrownOut = info.LastRun
	}

	*record = lastRun{magic: lastRunMagic, stage: uint32(StageStartup)}
	rp.VREG_AND_CHIP_RESET.SetBOD_VSEL(brownOutLevel)
	rp.VREG_AND_CHIP_RESET.SetBOD_EN(1)
	// Configure selects what a watchdog reset restarts, which Crash relies on before Start.
	machine.Watchdog.Configure(machine.WatchdogConfig{TimeoutMillis: WatchdogTimeout})
	return info
}

func StartWatchdog() {
	machine.Watchdog.Start()
}

// FeedWatchdog is called by the main loop each iteration; now is Millis.
func FeedWatchdog(now uint32) {
	record.uptime = now
	machine.Watchdog.Update()
}

func SetStage(stage Stage) {
	record.stage = uint32(stage)
}

// Recover is deferred at the top of every goroutine: a panic anywhere would otherwise halt the chip
// until the watchdog fires, with no trace of the cause. It stores the panic and restarts the Pico.
func Recover(where string) {
	value := recover()
	if value == nil {
		return
	}
	Crash(where + ": " + fmt.Sprint(value))
}

// Crash restarts the Pico at once; text is reported by ReadReset after the restart.
func Crash(text string) {
	println("panic:", text)
	record.kind = kindPanic
	record.length = uint32(copy(record.text[:], text))
	rp.WATCHDOG.SetCTRL_TRIGGER(1)
	for {
	}
}
