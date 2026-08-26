package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func BuildPayload() AgentPayload {

	return AgentPayload{
		Timestamp: time.Now(),
		Agent:     agentInfo,
		Stats:     currentStats,
		LastEvent: lastEvent,
	}
}

func BuildApiPayload() ApiPayload {

	return ApiPayload{

		Timestamp: time.Now(),

		Agent: ApiAgent{
			Username: agentInfo.Username,
			Computer: agentInfo.Computer,
		},

		Stats: ApiStats{

			MicroBreak: ApiBreakStats{
				Completed: currentStats.MicroBreak.NaturalTaken,
				Skipped:   currentStats.MicroBreak.Skipped,
				Postponed: currentStats.MicroBreak.Postponed,
				Delay:     currentStats.MicroBreak.Delayed,
			},

			RestBreak: ApiBreakStats{
				Completed: currentStats.RestBreak.NaturalTaken,
				Skipped:   currentStats.RestBreak.Skipped,
				Postponed: currentStats.RestBreak.Postponed,
				Delay:     currentStats.RestBreak.Delayed,
			},
		},

		Event: ApiEvent{
			Type:   lastEvent.BreakType,
			Action: lastEvent.EventType,
			Time:   lastEvent.Time,
		},
	}
}

func SendPayload(payload ApiPayload) {

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println("Error convirtiendo JSON:", err)
		return
	}

	resp, err := http.Post(
		apiURL,
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		log.Println("Error enviando payload:", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {

		log.Println("Payload enviado correctamente.")

	} else {

		log.Printf("El servidor respondió con código: %d\n", resp.StatusCode)

}
}