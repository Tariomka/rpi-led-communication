package services

import (
	"errors"
	"io"
	"log/slog"
	"strings"

	libcommon "github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/communication/uart"
	"github.com/Tariomka/rpi-led-communication/internal/common"
)

const (
	statusInterval     = 1500 // ms between Status polls
	statusTimeout      = 1000 // ms without a Status reply before the cube counts as not responding
	statusAfterCommand = 200  // ms after a command before its effect is polled
	statusAfterStartup = 1000 // ms after a cube restart or an upload before polling again
)

// Key and action codes of the cube's built-in shows; uploaded shows may remap keys.
const (
	KeyPreviousCartoon = 1
	KeyNextCartoon     = 2
	KeySpeedUp         = 3
	KeySpeedDown       = 4
	KeyPause           = 5
	KeyPlayModel       = 6
	KeyDisplayOff      = 7

	ActionReplay     = 1
	ActionSpeedReset = 5
)

// SerialPort is the UART the cube is connected to; reads must not block.
type SerialPort interface {
	io.Writer
	Buffered() int
	ReadByte() (byte, error)
}

// CubeState is the latest known state of the cube.
type CubeState struct {
	uart.Status
	Type       string // cube type from "T='...'"
	Responding bool
	StartedAt  uint32 // Millis of the last "Led Cube is starting up", 0 = not seen
}

// CubeService talks to the cube over UART: console commands, its text lines and show uploads.
// Run Poll from the main loop.
type CubeService struct {
	port   SerialPort
	logger *slog.Logger
	log    *LineLog
	reader uart.Reader
	upload *uart.Upload

	state   CubeState
	changed bool

	statusAt     uint32 // when the next Status is due
	statusSentAt uint32
	awaiting     bool // Status sent, reply not received yet

	uploading  bool
	uploadErr  error // overrides the upload's own error
	progressAt int   // report progress once this many bytes are stored, -1 = nothing to report
	lastAcked  int
	lastResult string

	OnLine     func(line string)
	OnProgress func(acked, total int)
	OnResult   func(ok bool, message string)
}

func NewCubeService(port SerialPort, log *LineLog, logger *slog.Logger) *CubeService {
	return &CubeService{
		port:       port,
		logger:     logger,
		log:        log,
		upload:     uart.NewUpload(port),
		progressAt: -1,
	}
}

func (this *CubeService) State() CubeState { return this.state }

// Changed reports whether the state or the upload progress changed since the last call.
func (this *CubeService) Changed() bool {
	changed := this.changed
	this.changed = false
	return changed
}

// Upload returns the state of the current or last upload and the result message of the last one.
func (this *CubeService) Upload() (state uart.UploadState, acked, total int, result string) {
	return this.upload.State(), this.upload.Acked(), this.upload.Total(), this.lastResult
}

func (this *CubeService) Uploading() bool { return this.uploading }

// Poll reads everything the cube sent and runs the timers; now is Millis.
func (this *CubeService) Poll(now uint32) {
	for this.port.Buffered() > 0 {
		b, err := this.port.ReadByte()
		if err != nil {
			break
		}

		switch item, data := this.reader.Feed(b, now); item {
		case uart.Line:
			if len(data) > 0 {
				this.handleLine(string(data), now)
			}
		case uart.Frame:
			if this.uploading {
				this.upload.Frame(data, now)
			}
		}
	}

	this.tickUpload(now)
	this.tickStatus(now)
}

// Command sends a console command line, e.g. "Key 2".
func (this *CubeService) Command(line string) error {
	if fields := strings.Fields(line); len(fields) > 0 && strings.EqualFold(fields[0], "ReBoot") {
		return common.ErrRebootCommand
	}

	command, err := uart.GetCommand(line)
	if err != nil {
		return err
	}
	return this.send(command)
}

// Key presses cube key n (1-7) like the physical key.
func (this *CubeService) Key(n int) error { return this.send(uart.GetCommandKey(n)) }

func (this *CubeService) Action(code int) error { return this.send(uart.GetCommandAction(code)) }

func (this *CubeService) Cartoon(index int) error { return this.send(uart.GetCommandCartoon(index)) }

func (this *CubeService) send(command []byte) error {
	if this.uploading {
		return libcommon.ErrUploadBusy
	}
	if _, err := this.port.Write(command); err != nil {
		return err
	}

	this.statusAt = common.Millis() + statusAfterCommand
	return nil
}

// BeginUpload starts storing a show of size bytes. Progress and the result are reported through
// OnProgress and OnResult.
func (this *CubeService) BeginUpload(size int) {
	if this.uploading {
		this.report(false, libcommon.ErrUploadBusy)
		return
	}

	now := common.Millis()
	// Any other error has already failed the upload and is reported by tickUpload.
	if err := this.upload.Begin(size, now); errors.Is(err, libcommon.ErrUploadSize) {
		this.report(false, err)
		return
	}

	this.uploading, this.uploadErr, this.progressAt, this.lastAcked = true, nil, -1, 0
	this.awaiting, this.changed = false, true
	this.logger.Info("Show upload started", "size", size)
	this.tickUpload(now)
}

// WriteUpload queues the next chunk of show data, which must start at offset.
func (this *CubeService) WriteUpload(offset uint32, data []byte) {
	switch {
	case !this.uploading:
		this.report(false, common.ErrNoUpload)
	case int(offset) != this.upload.Queued():
		this.upload.Abort()
		this.uploadErr = common.ErrUploadOffset
		this.tickUpload(common.Millis())
	default:
		now := common.Millis()
		this.upload.Write(data, now)
		this.progressAt = this.upload.Queued()
		this.tickUpload(now)
	}
}

// FinishUpload ends the upload once all data is stored.
func (this *CubeService) FinishUpload() {
	if !this.uploading {
		this.report(false, common.ErrNoUpload)
		return
	}

	now := common.Millis()
	this.upload.Finish(now)
	this.tickUpload(now)
}

// AbortUpload stops the upload without a result, e.g. when the desktop disconnects.
func (this *CubeService) AbortUpload() {
	if !this.uploading {
		return
	}

	this.upload.Abort()
	this.uploading, this.changed = false, true
	this.lastResult = "Aborted"
	this.statusAt = common.Millis()
	this.logger.Warn("Show upload aborted")
}

func (this *CubeService) handleLine(line string, now uint32) {
	if this.uploading {
		this.upload.Line(line)
	}

	event := uart.ParseLine(line)
	if event.Kind != uart.EventStatus {
		this.log.Add(SourceCube, line)
	}
	if this.OnLine != nil {
		this.OnLine(line)
	}

	previous := this.state
	switch event.Kind {
	case uart.EventStatus:
		this.state.Status = event.Status
		this.state.Responding, this.awaiting = true, false
	case uart.EventMode:
		this.state.Mode = event.Text
	case uart.EventCartoon:
		this.state.Cartoon = event.Values[0]
	case uart.EventSpeed:
		this.state.Speed = event.Values[0]
	case uart.EventPlayModel:
		this.state.PlayAll = event.Values[0] == 1
	case uart.EventPause:
		this.state.Pause = event.Values[0] == 1
	case uart.EventDisplayOff:
		this.state.DisplayOff = event.Values[0] == 1
	case uart.EventCubeType:
		this.state.Type = event.Text
	case uart.EventDevice:
		this.state.DeviceNum = event.Values[1]
	case uart.EventStartup:
		this.logger.Info("Cube restarted")
		this.state = CubeState{Responding: true, StartedAt: now}
		this.statusAt, this.awaiting = now+statusAfterStartup, false
	}
	if this.state != previous {
		this.changed = true
	}
}

func (this *CubeService) tickStatus(now uint32) {
	if this.uploading {
		return
	}

	if this.awaiting && now-this.statusSentAt >= statusTimeout {
		this.awaiting = false
		if this.state.Responding {
			this.state.Responding, this.changed = false, true
			this.logger.Warn("Cube is not responding")
		}
	}

	if int32(now-this.statusAt) >= 0 {
		if _, err := this.port.Write(uart.GetCommandStatus()); err != nil {
			this.logger.Error("Failed to write to the cube", "error", err)
		}
		this.statusSentAt, this.awaiting = now, true
		this.statusAt = now + statusInterval
	}
}

func (this *CubeService) tickUpload(now uint32) {
	if !this.uploading {
		return
	}

	this.upload.Tick(now)
	acked := this.upload.Acked()
	if acked != this.lastAcked {
		this.lastAcked, this.changed = acked, true
	}
	if this.progressAt >= 0 && acked >= this.progressAt {
		this.progressAt = -1
		if this.OnProgress != nil {
			this.OnProgress(acked, this.upload.Total())
		}
	}

	switch this.upload.State() {
	case uart.UploadDone:
		this.endUpload(now, nil)
	case uart.UploadFailed:
		err := this.uploadErr
		if err == nil {
			err = this.upload.Err()
		}
		this.endUpload(now, err)
	}
}

func (this *CubeService) endUpload(now uint32, err error) {
	this.uploading, this.changed = false, true
	this.statusAt = now + statusAfterStartup
	if err != nil {
		this.logger.Error("Show upload failed", "error", err)
	} else {
		this.logger.Info("Show upload OK")
	}
	this.report(err == nil, err)
}

func (this *CubeService) report(ok bool, err error) {
	this.lastResult = uart.LineUploadOK
	if err != nil {
		this.lastResult = err.Error()
	}
	if this.OnResult != nil {
		this.OnResult(ok, this.lastResult)
	}
}
