package runner

import (
	_ "embed"
	"io"
	"log/slog"
	"machine"
	"strings"

	"github.com/Tariomka/rpi-led-communication/internal/common"
)

//go:embed .env
var env string

type RunnerConfig struct {
	SSID          string // From .env file
	Password      string // From .env file
	ListenAddress string // From .env file
	Hostname      string // From .env file

	Logger *slog.Logger
}

func NewConfig() RunnerConfig {
	return readEmbededConfig()
}

func readEmbededConfig() RunnerConfig {
	config := RunnerConfig{}
	for _, line := range strings.Split(env, "\n") {
		split := strings.Split(line, "=")
		switch split[0] {
		case "SSID":
			config.SSID = split[1]
		case "Password":
			config.Password = split[1]
		case "ListenAddress":
			config.ListenAddress = split[1]
		case "Hostname":
			config.Hostname = split[1]
		default:
		}
	}

	return config
}

func (rc RunnerConfig) WithStructuredLogger() RunnerConfig {
	rc.Logger = slog.New(common.NewLogHandler(
		func(message string) { machine.USBCDC.Write([]byte(message + "\n")) },
		&slog.HandlerOptions{Level: slog.LevelInfo}))
	return rc
}

func (rc RunnerConfig) WithDebugLogger() RunnerConfig {
	rc.Logger = slog.New(slog.NewTextHandler(
		machine.USBCDC,
		&slog.HandlerOptions{Level: slog.LevelDebug - 2}))
	return rc
}

func (rc RunnerConfig) WithLogger(writer io.Writer, level slog.Level) RunnerConfig {
	rc.Logger = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level}))
	return rc
}
