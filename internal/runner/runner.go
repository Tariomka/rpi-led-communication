package runner

import (
	"encoding/json"
	"log/slog"
	"machine"
	"strconv"
	"strings"
	"time"

	"github.com/Tariomka/led-common-lib/pkg/communication/tcp"
	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/Tariomka/rpi-led-communication/internal/controller"
	"github.com/Tariomka/rpi-led-communication/internal/server"
	"github.com/Tariomka/rpi-led-communication/internal/services"
	"github.com/Tariomka/rpi-led-communication/internal/ui"
)

type Runner interface {
	Start()
	Stop()
}

type PicoRunner struct {
	config RunnerConfig
	picoW  *controller.PicoW
	cube   *services.CubeService
	ui     *ui.UI
	server *server.Server
	log    *services.LineLog
	logger *slog.Logger

	desktop bool
	history []services.LogLine
}

func NewRunner(config RunnerConfig) Runner {
	runner := &PicoRunner{
		config:  config,
		log:     &services.LineLog{},
		history: make([]services.LogLine, services.LogSize),
	}

	// Everything goes to USB; info and above also to the screen log and the desktop.
	runner.logger = slog.New(slog.NewMultiHandler(
		common.NewStructuredLogger(machine.USBCDC, slog.LevelDebug).Handler(),
		slog.NewTextHandler(lineWriter{runner}, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: dropTime,
		})))

	runner.picoW = controller.NewPicoW(controller.PicoConfig{
		SSID:     config.SSID,
		Password: config.Password,
		IP:       config.IP,
		Port:     config.Port,
		Hostname: config.Hostname,
	}, runner.logger)

	runner.cube = services.NewCubeService(runner.picoW.Cube, runner.log, runner.logger)
	runner.cube.OnLine = func(line string) { runner.send(tcp.NewCubeLine(line)) }
	runner.cube.OnProgress = func(acked, total int) {
		runner.send(tcp.NewUploadProgress(uint32(acked), uint32(total)))
	}
	runner.cube.OnResult = func(ok bool, message string) { runner.send(tcp.NewUploadResult(ok, message)) }

	runner.ui = ui.New(runner.picoW.Display, runner.cube, runner.log, config.Touch, runner.logger)
	runner.picoW.Display.SetYield(func() { runner.cube.Poll(common.Millis()) })
	return runner
}

func (this *PicoRunner) Start() {
	if !this.config.Touch.Valid() {
		calibration, _ := json.Marshal(this.ui.Calibrate())
		this.logger.Info(`Touch calibrated, add to config.json: "Touch": ` + string(calibration))
	}
	this.ui.Update(common.Millis())

	// Wi-Fi setup takes seconds (or over a minute when it fails); the cube and the screen keep running.
	links := make(chan desktopLink, 1)
	go func() {
		server, err := this.connectAndListen()
		links <- desktopLink{server, err}
	}()

	for {
		select {
		case link := <-links:
			this.startLink(link)
		default:
		}

		now := common.Millis()
		this.cube.Poll(now)
		this.handleRequest()
		this.ui.Update(now)
		time.Sleep(time.Millisecond)
	}
}

func (this *PicoRunner) Stop() {
	if this.server != nil {
		this.server.Stop()
	}
}

type desktopLink struct {
	server *server.Server
	err    error
}

func (this *PicoRunner) startLink(link desktopLink) {
	if link.err != nil {
		this.logger.Error("Wi-Fi setup failed, running without the desktop link", "error", link.err)
		this.ui.SetNetwork("not connected")
		return
	}

	this.server = link.server
	address := this.picoW.Addr().String() + ":" + strconv.Itoa(int(this.config.Port))
	this.logger.Info("Waiting for the desktop", "address", address)
	this.ui.SetNetwork(address)
	this.picoW.TurnLed(true)
	go this.server.Start()
}

func (this *PicoRunner) connectAndListen() (*server.Server, error) {
	if err := this.picoW.Connect(); err != nil {
		return nil, err
	}

	this.picoW.Blink(1)
	listener, err := this.picoW.GetListener()
	if err != nil {
		return nil, err
	}

	this.picoW.Blink(1)
	return server.NewServer(listener, this.logger)
}

func (this *PicoRunner) handleRequest() {
	if this.server == nil {
		return
	}

	select {
	case request := <-this.server.Requests():
		switch request.Kind {
		case server.Connected:
			this.desktop = true
			this.send(tcp.NewHello(this.config.Hostname))
			for _, line := range this.history[:this.log.Last(this.history)] {
				if line.Source == services.SourcePico {
					this.send(tcp.NewPicoLine(line.Text))
				} else {
					this.send(tcp.NewCubeLine(line.Text))
				}
			}
			this.ui.SetDesktop(true)
		case server.Disconnected:
			this.desktop = false
			this.cube.AbortUpload()
			this.ui.SetDesktop(false)
		case server.Received:
			this.handleMessage(request.Message)
		}
		this.server.Done()
	default:
	}
}

func (this *PicoRunner) handleMessage(message tcp.Message) {
	switch message.Type {
	case tcp.TypeHello:
		this.logger.Info("Desktop says hello", "name", message.Text())
	case tcp.TypeCommand:
		if err := this.cube.Command(message.Text()); err != nil {
			this.logger.Warn("Command rejected", "command", message.Text(), "error", err)
		}
	case tcp.TypeUploadBegin:
		size, err := message.Size()
		if err != nil {
			this.send(tcp.NewUploadResult(false, err.Error()))
			return
		}
		this.cube.BeginUpload(int(size))
	case tcp.TypeUploadChunk:
		offset, data, err := message.Chunk()
		if err != nil {
			this.cube.AbortUpload()
			this.send(tcp.NewUploadResult(false, err.Error()))
			return
		}
		this.cube.WriteUpload(offset, data)
	case tcp.TypeUploadEnd:
		this.cube.FinishUpload()
	case tcp.TypePing:
		this.send(tcp.NewPong())
	case tcp.TypePong:
	default:
		this.logger.Warn("Unexpected message", "type", message.Type)
	}
}

// send queues a message for the desktop, if one is connected.
func (this *PicoRunner) send(message tcp.Message) {
	if this.server != nil && this.desktop {
		this.server.Send(message)
	}
}

// lineWriter receives the Pico's log lines for the screen and the desktop.
type lineWriter struct{ runner *PicoRunner }

func (this lineWriter) Write(data []byte) (int, error) {
	line := shorten(strings.TrimRight(string(data), "\r\n"))
	this.runner.log.Add(services.SourcePico, line)
	this.runner.send(tcp.NewPicoLine(line))
	return len(data), nil
}

// shorten turns `level=INFO msg="text" key=value` into `INFO text key=value` to fit the screen.
func shorten(line string) string {
	level, rest, ok := strings.Cut(strings.TrimPrefix(line, "level="), " msg=")
	if !ok {
		return line
	}
	if quoted, err := strconv.QuotedPrefix(rest); err == nil {
		message, _ := strconv.Unquote(quoted)
		rest = message + rest[len(quoted):]
	}
	return level + " " + rest
}

func dropTime(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 && attr.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return attr
}
