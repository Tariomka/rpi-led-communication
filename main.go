package main

import (
	"log"

	"github.com/Tariomka/rpi-led-communication/internal/tcp"
)

func main() {
	server, err := tcp.NewServer(tcp.NewConfig())
	if err != nil {
		log.Fatalf("failed to start server: %v\n", err)
	}

	server.Start()
	defer server.Stop()
}
