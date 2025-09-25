package controller

import (
	"log/slog"
	"machine"
	"net"
	"strings"
	"time"

	"github.com/Tariomka/led-common-lib/pkg/network"
	"github.com/Tariomka/rpi-led-communication/internal/component"
)

const (
	maxRetries = 5
)

type Board interface {
	Connect() error // Connects to Wi-Fi. Returns an error if connection process fails.
	GetListener() (net.Listener, error)
	ReceiveFromUart()
	SentToUart(payload []byte)
	Blink(times uint)
	TurnLed(on bool)

	Screen()
}

type PicoConfig struct {
	SSID     string
	Password string
	IP       string
	Port     uint16
	Hostname string
}

type PicoW struct {
	wirelessChip *component.WirelessChip
	display      *component.Display

	logger        *slog.Logger
	uartProcessor *network.UartProcessor

	config PicoConfig
}

func NewPicoW(config PicoConfig, logger *slog.Logger) Board {
	uart := component.NewConfiguredUart(machine.UART0, machine.GP2)
	return &PicoW{
		wirelessChip: component.NewWirelessChip(logger),
		// display:        component.NewDisplay(),
		uartProcessor: network.NewUartProcessor(uart),
		logger:        logger,
		config:        config,
	}
}

func (this *PicoW) Connect() error {
	return this.wirelessChip.Connect(
		this.config.SSID,
		this.config.Password,
		this.config.IP,
		this.config.Hostname)
}

func (this *PicoW) GetListener() (net.Listener, error) {
	return this.wirelessChip.GetListener(this.config.Port)
}

func (this *PicoW) ReceiveFromUart() {
	// WORK IN PROGRESS
	go this.debugPing()
	// this.unprocessedReceiveFromUart()
	this.processedReceiveFromUart()
}

func (this *PicoW) SentToUart(payload []byte) {
	if err := this.uartProcessor.WriteBytes(payload); err != nil {
		this.logger.Error("Unexpected error when writing to UART", "error", err)
	}
}

func (this *PicoW) Blink(times uint) {
	for range times {
		this.wirelessChip.Blink()
	}
}

func (this *PicoW) TurnLed(on bool) {
	this.wirelessChip.TurnLed(on)
}

// TODO: Remove later
func (this *PicoW) processedReceiveFromUart() {
	retries := 0
	for {
		time.Sleep(1 * time.Second)
		dType, content, err := this.uartProcessor.Read()
		if err != nil {
			this.logger.Warn("Unexpected error while listening to UART", "error", err)
			if retries > maxRetries {
				this.logger.Error("Max retries reached, stopping listening to UART")
				this.uartProcessor.Desynchronize()
				break
			}

			retries++
			continue
		}

		retries = 0
		switch dType {
		case network.UartEmpty:
			this.logger.Debug("Received empty message", "content", content)
		case network.UartMessage:
			this.logger.Debug("Received message", "content", string(content))
		case network.UartBytes:
			this.logger.Debug("Received bytes", "content", content)
		case network.UartPing:
			this.uartProcessor.SendPong()
		}
	}
}

func (this *PicoW) unprocessedReceiveFromUart() {
	for {
		content, err := this.uartProcessor.ReadWithoutProcessing()
		if err != nil {
			this.logger.Warn("Unexpected error while listening to UART",
				"error", err,
				"content", string(content))
			continue
		}

		if len(content) == 0 {
			continue
		}

		this.logger.Debug("Received", "content", content, "content as string", string(content))
		if strings.Contains(string(content), "SGFuZHNoYWtl") {
			this.uartProcessor.Synchronize()
			return
		}
	}
}

func (this *PicoW) debugPing() {
	for {
		select {
		case <-time.After(5 * time.Second):
			this.uartProcessor.WriteMessage("Hello from RPi!")
		}
	}
}

func (this *PicoW) Screen() {
	if this.display == nil {
		this.logger.Debug("Initializing screen...")
		this.display = component.NewDisplay()
	}

	for {
		this.logger.Debug("Drawing on screen...")
		this.display.Draw()
	}
}
