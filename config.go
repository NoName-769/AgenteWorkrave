package main

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

var (
	agentID   string
	serverURL string
	wsURL     string
	apiURL    string
)

const defaultServerURL = "https://workrave-api.onrender.com"

func getAgentDir() string {

	exePath, err := os.Executable()

	if err != nil {
		return "."

	}

	return filepath.Dir(exePath)
}

func getConfigPath() string {

	return filepath.Join(
		getAgentDir(),
		"agent.ini",
	)
}

func loadConfig() {

	iniPath := getConfigPath()

	log.Printf("Config path: %s", iniPath)

	serverURL = defaultServerURL
	apiURL = serverURL + "/api/workrave"
	hostname, err := os.Hostname()

	if err != nil || hostname == "" {
		hostname = "unknown_agent"
	}

	agentID = hostname

	if _, err := os.Stat(iniPath); os.IsNotExist(err) {

		cfg := ini.Empty()

		sec, err := cfg.NewSection("agent")

		if err != nil {
			log.Println("Error creando sección agent:", err)
		} else {

			sec.Key("server_url").SetValue(defaultServerURL)
			sec.Key("api_url").SetValue(
				defaultServerURL + "/api/workrave",
			)

			sec.Key("agent_id").SetValue(agentID)

			if err := cfg.SaveTo(iniPath); err != nil {

				log.Println(
					"Error creando agent.ini:",
					err,
				)

			} else {

				log.Printf(
					"Archivo de configuración creado: %s",
					iniPath,
				)
			}
		}
	}

	cfg, err := ini.Load(iniPath)

	if err != nil {

		log.Println(
			"Error cargando agent.ini:",
			err,
		)

	} else {

		sec := cfg.Section("agent")

		if val := sec.Key("server_url").String(); val != "" {

			serverURL = val

		}

		if val := sec.Key("api_url").String(); val != "" {

			apiURL = val

		}

		if val := sec.Key("agent_id").String(); val != "" && val != "unknown_agent" {
			agentID = val
		}
	}

	if envAgent := os.Getenv("AGENT_ID"); envAgent != "" {

		agentID = envAgent

	}

	if envServer := os.Getenv("SERVER_URL"); envServer != "" {

		serverURL = envServer

	}

	if envAPI := os.Getenv("API_URL"); envAPI != "" {

		apiURL = envAPI

	}

	/*
		==================================================
		VALORES FINALES
		==================================================
	*/

	if serverURL == "" {

		serverURL = defaultServerURL

	}

	if apiURL == "" {

		apiURL = serverURL + "/api/workrave"

	}

	wsURL = os.Getenv("WS_URL")

	if wsURL == "" {

		if len(serverURL) > 7 &&
			serverURL[:7] == "http://" {

			wsURL =
				"ws://" +
					serverURL[7:] +
					"/ws"

		} else if len(serverURL) > 8 &&
			serverURL[:8] == "https://" {

			wsURL =
				"wss://" +
					serverURL[8:] +
					"/ws"

		} else {

			wsURL =
				"wss://" +
					serverURL +
					"/ws"

		}
	}

	log.Printf(
		"Agent ID: %s",
		agentID,
	)

	log.Printf(
		"Server URL: %s",
		serverURL,
	)

	log.Printf(
		"API URL: %s",
		apiURL,
	)

	log.Printf(
		"WebSocket URL: %s",
		wsURL,
	)
}
