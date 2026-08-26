package main

import (
	"strings"
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

func getWorkraveProcess() (*process.Process, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	for _, p := range procs {
		name, err := p.Name()
		if err == nil && strings.EqualFold(name, "workrave.exe") {
			return p, nil
		}
	}

	return nil, process.ErrorProcessNotRunning
}
func monitorWorkraveWindows() {

	last := ""

	for {

		window := getActiveWorkraveWindow()

		current := ""

		if window != nil {
			current = window.Title
		}

		if current != last {

			if current == "" {
				log.Println("No hay ventana de descanso")
			} else {
				log.Printf("Ventana detectada: %s", current)
			}

			last = current
		}

		time.Sleep(250 * time.Millisecond)
	}
}