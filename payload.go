package main

import "time"

type AgentPayload struct {
	Timestamp time.Time      `json:"timestamp"`
	Agent      AgentInfo      `json:"agent"`
	Stats      *TodayStats    `json:"stats"`
	LastEvent  WorkraveEvent  `json:"lastEvent"`
}