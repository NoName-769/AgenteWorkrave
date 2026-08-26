package main

import "time"

type ApiPayload struct {
	Timestamp time.Time    `json:"timestamp"`
	Agent      ApiAgent     `json:"agent"`
	Stats      ApiStats     `json:"stats"`
	Event      ApiEvent     `json:"event"`
}

type ApiAgent struct {
	Username string `json:"username"`
	Computer string `json:"computer"`
}

type ApiStats struct {
	MicroBreak ApiBreakStats `json:"microBreak"`
	RestBreak  ApiBreakStats `json:"restBreak"`
}

type ApiBreakStats struct {
	Completed int `json:"completed"`
	Skipped   int `json:"skipped"`
	Postponed int `json:"postponed"`
	Delay     int `json:"delay"`
}

type ApiEvent struct {
	Type   string    `json:"type"`
	Action string    `json:"action"`
	Time   time.Time `json:"time"`
}