package runner

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"machine"

	"github.com/Tariomka/rpi-led-communication/internal/common"
)

//go:embed config.json
var config []byte

type RunnerConfig struct {
	SSID     string `json:"SSID"`
	Password string `json:"Password"`
	IP       string `json:"IP,omitempty"`
	Port     uint16 `json:"Port"`
	Hostname string `json:"Hostname"`

	Logger *slog.Logger
}

func NewConfig() RunnerConfig {
	return readEmbededConfig()
}

func readEmbededConfig() RunnerConfig {
	var rc RunnerConfig
	if err := json.Unmarshal(config, &rc); err != nil {
		fmt.Printf("failed to parse config: %s\n", err.Error())
	}
	return rc
}

func (this RunnerConfig) WithStructuredLogger() RunnerConfig {
	this.Logger = common.NewStructuredLogger(machine.USBCDC, slog.LevelInfo)
	return this
}

func (this RunnerConfig) WithDebugLogger() RunnerConfig {
	this.Logger = common.NewSimpleLogger(machine.USBCDC, slog.LevelDebug-2)
	return this
}

func (this RunnerConfig) WithLogger(writer io.Writer, level slog.Level) RunnerConfig {
	this.Logger = common.NewSimpleLogger(writer, level)
	return this
}
