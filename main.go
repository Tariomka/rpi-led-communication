package main

import (
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/runner"
)

func main() {
	time.Sleep(time.Second) // Some time for monitoring to start
	runner := runner.NewRunner(runner.NewConfig().WithStructuredLogger())
	runner.Start()
	defer runner.Stop()
}
