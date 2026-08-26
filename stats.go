package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type BreakStats struct {
	Requested    int
	Taken        int
	NaturalTaken int
	Skipped      int
	Postponed    int
	UniqueTaken  int
	Delayed      int
}

type TodayStats struct {
	MicroBreak BreakStats
	RestBreak  BreakStats
	DailyLimit BreakStats
}

type WorkraveEvent struct {
	BreakType string
	EventType string
	OldValue  int
	NewValue  int
	Time      time.Time
}

func ReadTodayStats() (*TodayStats, error) {

	stats := &TodayStats{}

	path := os.Getenv("APPDATA") + "\\Workrave\\todaystats"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		parts := strings.Fields(line)

		if len(parts) == 0 {
			continue
		}

		if parts[0] != "B" {
			continue
		}

		breakID := parts[1]

		requested, _ := strconv.Atoi(parts[3])
		taken, _ := strconv.Atoi(parts[4])
		naturalTaken, _ := strconv.Atoi(parts[5])
		skipped, _ := strconv.Atoi(parts[6])
		postponed, _ := strconv.Atoi(parts[7])
		uniqueTaken, _ := strconv.Atoi(parts[8])
		delayed, _ := strconv.Atoi(parts[9])

		current := BreakStats{
			Requested:    requested,
			Taken:        taken,
			NaturalTaken: naturalTaken,
			Skipped:      skipped,
			Postponed:    postponed,
			UniqueTaken:  uniqueTaken,
			Delayed:      delayed,
		}

		switch breakID {

		case "0":
			stats.MicroBreak = current

		case "1":
			stats.RestBreak = current

		case "2":
			stats.DailyLimit = current
		}
	}

	return stats, nil
}

func PrintTodayStats(stats *TodayStats) {

	log.Println("=================================")
	log.Println("      ESTADÍSTICAS WORKRAVE")
	log.Println("=================================")

	printBreak("MICRO-PAUSA", stats.MicroBreak)
	printBreak("DESCANSO", stats.RestBreak)
	printBreak("LÍMITE DIARIO", stats.DailyLimit)
}

func printBreak(title string, b BreakStats) {

	log.Println("")
	log.Println(title)
	log.Println("---------------------------------")
	log.Printf("Avisos:              %d", b.Requested)
	log.Printf("Tomadas:             %d", b.Taken)
	log.Printf("Completadas:           %d", b.NaturalTaken)
	log.Printf("Saltadas:            %d", b.Skipped)
	log.Printf("Aplazadas:           %d", b.Postponed)
	log.Printf("Pedidos y tomadas:   %d", b.UniqueTaken)
	log.Printf("Tiempo retrasado:    %d s", b.Delayed)
}

func compareBreak(name string, previous, current BreakStats) {

	if current.Skipped > previous.Skipped {

		handleEvent(WorkraveEvent{
			BreakType: name,
			EventType: "Skipped",
			OldValue:  previous.Skipped,
			NewValue:  current.Skipped,
			Time:      time.Now(),
		})
	}

	if current.NaturalTaken > previous.NaturalTaken {

		handleEvent(WorkraveEvent{
			BreakType: name,
			EventType: "Taken",
			OldValue:  previous.NaturalTaken,
			NewValue:  current.NaturalTaken,
			Time:      time.Now(),
		})
	}

	if current.Postponed > previous.Postponed {

		handleEvent(WorkraveEvent{
			BreakType: name,
			EventType: "Postponed",
			OldValue:  previous.Postponed,
			NewValue:  current.Postponed,
			Time:      time.Now(),
		})
	}
}

func handleEvent(event WorkraveEvent) {

	lastEvent = event

	log.Println("=================================")
	log.Println("EVENTO WORKRAVE")
	log.Println("=================================")
	log.Printf("Tipo de descanso : %s", event.BreakType)
	log.Printf("Evento           : %s", event.EventType)
	log.Printf("Valor anterior   : %d", event.OldValue)
	log.Printf("Valor nuevo      : %d", event.NewValue)
	log.Printf("Hora             : %s", event.Time.Format("15:04:05"))
	log.Println("")
	payload := BuildApiPayload()
	lastPayload = payload
	SendPayload(payload)
}

func MonitorTodayStats() {

	var previous *TodayStats

	for {

		current, err := ReadTodayStats()
		currentStats = current
		if err != nil {
			log.Println("Error leyendo estadísticas:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if previous == nil {

			log.Println("Primera lectura de estadísticas.")
			PrintTodayStats(current)

			previous = current
			enviarEstadisticas(current)

			time.Sleep(1 * time.Second)
			continue
		}

		compareBreak("Micro-pausa", previous.MicroBreak, current.MicroBreak)

		compareBreak("Descanso", previous.RestBreak, current.RestBreak)

		compareBreak("Límite diario", previous.DailyLimit, current.DailyLimit)

		previous = current
		enviarEstadisticas(current)

		time.Sleep(5 * time.Second)
	}
}
