package runner

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/Tariomka/rpi-led-communication/internal/ui"
)

//go:embed config.json
var config []byte

type RunnerConfig struct {
	SSID     string         `json:"SSID"`
	Password string         `json:"Password"`
	IP       string         `json:"IP,omitempty"`
	Port     uint16         `json:"Port"`
	Hostname string         `json:"Hostname"`
	Touch    ui.Calibration `json:"Touch"` // calibrated on start when missing
}

func NewConfig() RunnerConfig {
	return readEmbededConfig()
}

func readEmbededConfig() RunnerConfig {
	var rc RunnerConfig
	if err := json.Unmarshal(config, &rc); err != nil {
		fmt.Printf("[RUNNER_CONFIG] Failed to parse config: %s\n", err.Error())
	}
	return rc
}
