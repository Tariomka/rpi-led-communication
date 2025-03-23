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
	IP       string `json:"IP"`
	Port     int    `json:"Port"`
	Hostname string `json:"Hostname"`

	Logger *slog.Logger
}

func NewConfig() RunnerConfig {
	return readEmbededConfig()
}

func readEmbededConfig() RunnerConfig {
	var rc RunnerConfig
	if err := json.Unmarshal(config, &rc); err != nil {
		fmt.Errorf("failed to parse config", "err", err.Error())
	}
	return rc
}

func (rc RunnerConfig) WithStructuredLogger() RunnerConfig {
	rc.Logger = common.NewStructuredLogger(machine.USBCDC, slog.LevelInfo)
	return rc
}

func (rc RunnerConfig) WithDebugLogger() RunnerConfig {
	rc.Logger = common.NewSimpleLogger(machine.USBCDC, slog.LevelDebug-2)
	return rc
}

func (rc RunnerConfig) WithLogger(writer io.Writer, level slog.Level) RunnerConfig {
	rc.Logger = common.NewSimpleLogger(writer, level)
	return rc
}
