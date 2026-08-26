package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var currentStats *TodayStats
var lastEvent WorkraveEvent
var agentInfo AgentInfo
var lastPayload ApiPayload

func startDashboard() {

	http.HandleFunc("/", dashboardHandler)
	http.HandleFunc("/style.css", styleHandler)
	http.HandleFunc("/script.js", scriptHandler)
	http.HandleFunc("/api/stats", statsHandler)

	http.HandleFunc("/api/workrave/status", workraveStatusHandler)
	http.HandleFunc("/api/workrave/open", workraveOpenHandler)
	http.HandleFunc("/api/workrave/close", workraveCloseHandler)
	http.HandleFunc("/api/workrave/stats", workraveStatsHandler)

	fmt.Println("Dashboard disponible en http://localhost:8082")

	if err := http.ListenAndServe(":8082", nil); err != nil {
		fmt.Println(err)
	}
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, "web/index.html")
}

func styleHandler(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, "web/style.css")
}

func scriptHandler(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, "web/script.js")
}

func statsHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(lastPayload)

}

func workraveStatusHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	status := WorkraveStatus()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"running": status,
	})
}

func workraveOpenHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	err := OpenWorkrave()

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Workrave abierto correctamente",
	})
}

func workraveCloseHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	err := CloseWorkrave()

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Workrave cerrado correctamente",
	})
}

func workraveStatsHandler(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    if currentStats == nil {
        http.Error(w, "Estadísticas no disponibles", http.StatusServiceUnavailable)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "microBreak": map[string]interface{}{
            "completed": currentStats.MicroBreak.NaturalTaken,
            "skipped":   currentStats.MicroBreak.Skipped,
            "postponed": currentStats.MicroBreak.Postponed,
            "delay":     currentStats.MicroBreak.Delayed,
        },
        "restBreak": map[string]interface{}{
            "completed": currentStats.RestBreak.NaturalTaken,
            "skipped":   currentStats.RestBreak.Skipped,
            "postponed": currentStats.RestBreak.Postponed,
            "delay":     currentStats.RestBreak.Delayed,
        },
    })
}