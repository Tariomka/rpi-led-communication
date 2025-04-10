package main

import "github.com/Tariomka/rpi-led-communication/internal/runner"

func main() {
	runner := runner.NewRunner(runner.NewConfig().WithStructuredLogger())
	runner.Start()
	defer runner.Stop()
}
