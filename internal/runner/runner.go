package runner

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/Tariomka/rpi-led-communication/internal/controller"
	"github.com/Tariomka/rpi-led-communication/internal/tcp"
)

//go:embed .env
var env string

type RunnerConfig struct {
	SSID          string
	Password      string
	ListenAddress string
}

func NewConfig() RunnerConfig {
	return readConfig()
}

func readConfig() RunnerConfig {
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
		default:
		}
	}

	return config
}

type Runner interface {
	Start()
	Stop()
}

type PicoRunner struct {
	PicoW        controller.Board
	LayoutWorker controller.LayoutWorker
	Server       tcp.Server
}

func NewRunner(config RunnerConfig) (Runner, error) {
	fmt.Printf("Data: %s\n", config)

	// server, err := tcp.NewServer(tcp.ServerConfig{Address: config.ListenAddress})
	// if err != nil {
	// 	return nil, err
	// }

	return &PicoRunner{
		PicoW:        controller.NewPicoW(),
		LayoutWorker: &controller.LedLayout{},
		// Server:       server,
	}, nil
}

func (pr *PicoRunner) Start() {
	// pr.Server.Start()
	testOut()
}

func (pr *PicoRunner) Stop() {
	// defer pr.Server.Stop()
}
