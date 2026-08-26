package main

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

func monitorDBus() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("no se pudo conectar al Session Bus: %w", err)
	}
	defer conn.Close()

	fmt.Println("[DBus] Conectado al Session Bus")

	return nil
}