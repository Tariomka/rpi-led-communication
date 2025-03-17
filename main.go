package main

import (
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/runner"
)

func main() {
	time.Sleep(time.Second) // Some time for monitoring to start
	runner, err := runner.NewRunner(runner.NewConfig())
	if err != nil {
		panic(err)
	}

	runner.Start()
	defer runner.Stop()
}
