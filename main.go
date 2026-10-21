// Command rpi-led-communication runs the project's tasks: building, flashing and monitoring the
// Pico W firmware. Run it from the repository root:
//
//	go run . <task> [arguments]
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	firmwarePkg = "./cmd/rpi_led_com"
	target      = "pico-w"
	configFile  = "internal/runner/config.json"
)

var (
	uf2File    = filepath.Join("bin", "rpi_led_com.uf2")
	flashFlags = []string{"-gc=precise", "-opt=2", "-no-debug", "-size", "full"}
)

type task struct {
	name, usage, help string
	run               func(args []string) error
}

var tasks []task

func init() {
	tasks = []task{
		{"build", "", "TinyGo build into " + uf2File, build},
		{"monitor", "[port]", "Prints the Pico W's USB serial log", monitor},
	}
}

func main() {
	if len(os.Args) > 1 {
		for _, task := range tasks {
			if task.name == os.Args[1] {
				if err := task.run(os.Args[2:]); err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					os.Exit(1)
				}
				return
			}
		}
	}

	fmt.Println("Usage: go run . <task> [arguments]   (port: e.g. COM5 or /dev/ttyACM0, detected when omitted)")
	fmt.Println()
	for _, task := range tasks {
		fmt.Printf("  %-8s %-7s %s\n", task.name, task.usage, task.help)
	}
	os.Exit(2)
}

// execute runs a program in the foreground with the console attached.
func execute(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	return command.Run()
}

// portArgs returns the -port flag when a port is given.
func portArgs(args []string) []string {
	if len(args) > 0 {
		return []string{"-port", args[0]}
	}
	return nil
}

// ensureConfig creates an empty config.json when it is missing; the firmware embeds it.
func ensureConfig() error {
	file, err := os.OpenFile(configFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func build(args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}
	if err := os.MkdirAll("bin", 0o755); err != nil {
		return err
	}

	fmt.Println("Building " + firmwarePkg + " ...")
	err := execute("tinygo", "build", "-o", uf2File, "-target", target, firmwarePkg)
	if err == nil {
		fmt.Println("Built " + uf2File)
	}
	return err
}

func monitor(args []string) error {
	return execute("tinygo", append([]string{"monitor", "-target", target}, portArgs(args)...)...)
}
