package main

import (
	"os"
)

type AgentInfo struct {
	Username string `json:"username"`
	Computer string `json:"computer"`
}

func GetAgentInfo() AgentInfo {

	return AgentInfo{
		Username: os.Getenv("USERNAME"),
		Computer: os.Getenv("COMPUTERNAME"),
	}
}