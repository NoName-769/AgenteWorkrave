package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"sort"
)

func byteIndex(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

type Telemetry struct {
	AgentID   string `json:"agentId"`
	Timestamp int64  `json:"timestamp"`
	CPUUsage  string `json:"cpuUsage"`
	Memory    string `json:"memory"`
	Disk      string `json:"disk"`
	Network   string `json:"network"`
	PublicIP  string `json:"publicIp"`
	Lookup    string `json:"lookup"`
	Status    string `json:"status"`
}

func getTailscaleIP() string {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if strings.HasPrefix(i.Name, "tailscale") || strings.HasPrefix(strings.ToLower(i.Name), "tailscale") {
				addrs, _ := i.Addrs()
				for _, addr := range addrs {
					var ip net.IP
					switch v := addr.(type) {
					case *net.IPNet:
						ip = v.IP
					case *net.IPAddr:
						ip = v.IP
					}
					if ip.To4() != nil && strings.HasPrefix(ip.String(), "100.") {
						return ip.String()
					}
				}
			}
		}
	}
	// Fallback to checking all interfaces for 100.x.x.x
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil && strings.HasPrefix(ipnet.IP.String(), "100.") {
				return ipnet.IP.String()
			}
		}
	}
	return "0.0.0.0"
}

func enviarHeartbeat() {

	for {

		running := isWorkraveRunning()
		threeCX := is3CXRunning()

		payload := map[string]interface{}{
			"running": running,
			"threeCX": threeCX,
		}

		body, err := json.Marshal(payload)

		if err != nil {
			log.Println("Error preparando heartbeat:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		url := serverURL + "/api/agents/" + agentID + "/heartbeat"

		req, err := http.NewRequest(
			"POST",
			url,
			bytes.NewBuffer(body),
		)

		if err != nil {
			log.Println("Error creando heartbeat:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)

		if err != nil {
			log.Println("Error enviando heartbeat:", err)
		} else {
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				log.Println("Heartbeat enviado correctamente")
			} else {
				log.Println(
					"Heartbeat respondió:",
					resp.StatusCode,
				)
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func registerAgent() {

	host, err := os.Hostname()

	if err != nil {
		host = "UNKNOWN"
	}

	tailscaleIP := getTailscaleIP()

	if tailscaleIP == "0.0.0.0" {
		log.Println("No se encontró IP de Tailscale.")
		return
	}

	payload := map[string]interface{}{
		"agentId":     agentID,
		"hostname":    host,
		"tailscaleIp": tailscaleIP,
		"port":        8082,
	}

	data, err := json.Marshal(payload)

	if err != nil {
		log.Println("Error preparando registro del agente:", err)
		return
	}

	resp, err := http.Post(
		serverURL+"/api/agents/register",
		"application/json",
		bytes.NewBuffer(data),
	)

	if err != nil {
		log.Println("No se pudo registrar el agente:", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {

		log.Printf(
			"Agente registrado: %s | Hostname: %s | Tailscale: %s",
			agentID,
			host,
			tailscaleIP,
		)

	} else {

		log.Printf(
			"Error registrando agente: %s",
			resp.Status,
		)
	}
}

func monitorAgentRegistration() {

	registerAgent()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		registerAgent()
	}
}

func enviarEstadisticas(stats interface{}) {

	payload := map[string]interface{}{
		"stats": stats,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		log.Println("Error preparando estadísticas:", err)
		return
	}

	url := serverURL + "/api/agents/" + agentID + "/stats"

	req, err := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer(body),
	)

	if err != nil {
		log.Println("Error creando petición de estadísticas:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Println("Error enviando estadísticas:", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("Estadísticas enviadas correctamente")
	} else {
		log.Println("Estadísticas respondieron:", resp.StatusCode)
	}
}

type procInfo struct {
	Pid  int32
	Name string
	Mem  float32
	Cpu  float64
}

func getTopProcessesDetails() []map[string]interface{} {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}
	var infos []procInfo
	for _, p := range procs {
		m, _ := p.MemoryPercent()
		c, _ := p.CPUPercent()
		n, _ := p.Name()
		if m > 0 || c > 0 {
			infos = append(infos, procInfo{Pid: p.Pid, Name: n, Mem: m, Cpu: c})
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Mem > infos[j].Mem
	})
	limit := 15
	if limit > len(infos) {
		limit = len(infos)
	}

	var res []map[string]interface{}
	for _, info := range infos[:limit] {
		res = append(res, map[string]interface{}{
			"pid":  info.Pid,
			"name": info.Name,
			"mem":  fmt.Sprintf("%.1f%%", info.Mem),
			"cpu":  fmt.Sprintf("%.1f%%", info.Cpu),
		})
	}
	return res
}

var permitirAperturaWorkrave = true

func is3CXRunning() bool {

	procs, err := process.Processes()

	if err != nil {
		return false
	}

	for _, p := range procs {

		name, err := p.Name()

		if err == nil && strings.EqualFold(name, "3CXSoftphone.exe") {
			return true
		}
	}

	return false
}

func monitorWorkrave() {

	for {

		if is3CXRunning() {

			log.Println(
				"3CX está activo. Workrave permanece cerrado.",
			)

			time.Sleep(1 * time.Minute)
			continue
		}

		if !permitirAperturaWorkrave {
			time.Sleep(1 * time.Minute)
			continue
		}

		if !WorkraveStatus() {

			log.Println(
				"Workrave está cerrado. Intentando abrir...",
			)

			err := OpenWorkrave()

			if err != nil {

				log.Println(
					"Error abriendo Workrave:",
					err,
				)

			} else {

				log.Println(
					"Orden de apertura enviada",
				)

				time.Sleep(5 * time.Second)

				if WorkraveStatus() {

					log.Println(
						"Workrave iniciado correctamente",
					)

				} else {

					log.Println(
						"Workrave todavía no aparece como iniciado",
					)
				}
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

// Revisar si WR esta funcionando
func isWorkraveRunning() bool {
	procs, err := process.Processes()
	if err != nil {
		return false
	}

	for _, p := range procs {
		name, err := p.Name()
		if err == nil && strings.EqualFold(name, "workrave.exe") {
			return true
		}
	}

	return false
}

func monitor3CX() {

	var threeCXAnterior bool
	var workraveEstabaAbierto bool

	for {

		threeCXActivo := is3CXRunning()

		if threeCXActivo && !threeCXAnterior {

			log.Println("3CX detectado: ACTIVO")
			permitirAperturaWorkrave = false

			workraveEstabaAbierto = isWorkraveRunning()

			if workraveEstabaAbierto {

				log.Println("3CX activo. Cerrando Workrave...")

				err := CloseWorkrave()

				if err != nil {
					log.Println(
						"Error cerrando Workrave por 3CX:",
						err,
					)
				} else {
					log.Println(
						"Workrave cerrado debido a 3CX activo",
					)
				}

			} else {

				log.Println(
					"3CX activo, pero Workrave ya estaba cerrado",
				)
			}
		}

		if !threeCXActivo && threeCXAnterior {

			log.Println("3CX detectado: CERRADO")

			if workraveEstabaAbierto {

				if !isWorkraveRunning() {

					permitirAperturaWorkrave = true

					log.Println(
						"3CX cerrado. Restaurando Workrave...",
					)

					err := OpenWorkrave()

					if err != nil {

						log.Println(
							"Error restaurando Workrave:",
							err,
						)

					} else {

						log.Println(
							"Workrave restaurado correctamente",
						)
					}

				} else {

					log.Println(
						"Workrave ya está abierto",
					)
				}

			} else {

				permitirAperturaWorkrave = true

				log.Println(
					"3CX cerrado. Restaurando Workrave...",
				)
			}

			workraveEstabaAbierto = false
		}

		threeCXAnterior = threeCXActivo

		time.Sleep(1 * time.Minute)
	}
}

func getDockerContainersDetails() []map[string]interface{} {
	cmd := exec.Command("docker", "ps", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var res []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		var c map[string]interface{}
		json.Unmarshal([]byte(l), &c)
		res = append(res, c)
	}
	return res
}

func getNetworksDetails() []map[string]interface{} {
	var res []map[string]interface{}
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		addrs, _ := i.Addrs()
		var ips []string
		for _, a := range addrs {
			ips = append(ips, a.String())
		}
		res = append(res, map[string]interface{}{
			"name": i.Name,
			"ips":  strings.Join(ips, ", "),
			"mac":  i.HardwareAddr.String(),
		})
	}
	return res
}

func getWifiDetails() string {
	if runtime.GOOS == "windows" {
		out, _ := exec.Command("netsh", "wlan", "show", "networks").CombinedOutput()
		return string(out)
	} else {
		out, _ := exec.Command("nmcli", "d", "wifi").CombinedOutput()
		return string(out)
	}
}

func getDockerCount() string {
	cmd := exec.Command("docker", "ps", "-q")
	out, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "0"
	}
	return fmt.Sprintf("%d", len(lines))
}

func getSeaweedfsDetails() []map[string]interface{} {
	if runtime.GOOS == "windows" {
		return nil
	}
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager", "--no-legend", "sea*")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var res []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) >= 4 {
			res = append(res, map[string]interface{}{
				"unit":        fields[0],
				"load":        fields[1],
				"active":      fields[2],
				"sub":         fields[3],
				"description": strings.Join(fields[4:], " "),
			})
		}
	}
	return res
}

func sendTelemetry() {
	for {
		// CPU (measures over 1 second)
		cpuPercent, _ := cpu.Percent(time.Second, false)
		coreCount, _ := cpu.Counts(true)
		var cUsage string
		if len(cpuPercent) > 0 {
			cUsage = fmt.Sprintf("%.1f%% (%d Cores)", cpuPercent[0], coreCount)
		} else {
			cUsage = fmt.Sprintf("0%% (%d Cores)", coreCount)
		}

		// Mem
		vMem, _ := mem.VirtualMemory()
		var memUsage string
		if vMem != nil {
			memUsage = fmt.Sprintf("%.1fGB/%.1fGB", float64(vMem.Used)/1024/1024/1024, float64(vMem.Total)/1024/1024/1024)
		}

		// Disk
		diskPath := "/"
		if runtime.GOOS == "windows" {
			diskPath = "C:\\"
		}
		dUsage, _ := disk.Usage(diskPath)
		var diskStr string
		if dUsage != nil {
			diskStr = fmt.Sprintf("%.1fGB/%.1fGB (%.1f%%)", float64(dUsage.Used)/1024/1024/1024, float64(dUsage.Total)/1024/1024/1024, dUsage.UsedPercent)
		}

		// Network / Tailscale IP
		tsIP := getTailscaleIP()
		networkStatus := "Online (Tailscale)"
		if tsIP == "0.0.0.0" {
			networkStatus = "Online (No Tailscale)"
		}

		dockerCount := getDockerCount()
		host, _ := os.Hostname()

		data := Telemetry{
			AgentID:   agentID,
			Timestamp: time.Now().Unix(),
			CPUUsage:  cUsage,
			Memory:    memUsage,
			Disk:      diskStr,
			Network:   networkStatus,
			PublicIP:  tsIP,
			Lookup:    host + " (Docker: " + dockerCount + ")",
			Status:    "Healthy",
		}

		payload, err := json.Marshal(data)
		if err == nil {
			resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payload))
			if err != nil {
				log.Printf("Failed to send telemetry: %v", err)
			} else {
				resp.Body.Close()
				log.Printf("Sent telemetry for %s", agentID)
			}
		}
		time.Sleep(9 * time.Second)
	}
}

func getScreenBase64() string {
	start := time.Now()
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return ""
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	// Using JPEG with quality 60 for optimal streaming speed/bandwidth
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 60}); err != nil {
		return ""
	}

	res := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	log.Printf("Frame captured and encoded in %v", time.Since(start))
	return res
}

func connectWS() {
	for {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			log.Printf("WS dial error: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("WS connected!")

		// Register
		regMsg := map[string]string{
			"type":    "register_agent",
			"agentId": agentID,
		}

		c.WriteJSON(regMsg)
		c.WriteJSON(map[string]interface{}{
			"type":    "workrave_status",
			"agentId": agentID,
			"running": isWorkraveRunning(),
		})

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("WS read error:", err)
				c.Close()
				break
			}

			var msg map[string]interface{}

			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			mType, _ := msg["type"].(string)
			requester, _ := msg["requester"].(string)

			if mType == "request_screen" {

				screenData := getScreenBase64()

				if screenData != "" {
					c.WriteJSON(map[string]string{
						"type":      "response_screen",
						"agentId":   agentID,
						"image":     screenData,
						"requester": requester,
					})
				}

			} else if mType == "request_processes" {

				c.WriteJSON(map[string]interface{}{
					"type":      "response_processes",
					"agentId":   agentID,
					"requester": requester,
					"data":      getTopProcessesDetails(),
				})

			} else if mType == "request_docker" {

				c.WriteJSON(map[string]interface{}{
					"type":      "response_docker",
					"agentId":   agentID,
					"requester": requester,
					"data":      getDockerContainersDetails(),
				})

			} else if mType == "restart_docker" {

				cid, _ := msg["containerId"].(string)

				exec.Command("docker", "restart", cid).Run()

				c.WriteJSON(map[string]interface{}{
					"type":      "response_docker",
					"agentId":   agentID,
					"requester": requester,
					"data":      getDockerContainersDetails(),
				})

			} else if mType == "request_networks" {

				c.WriteJSON(map[string]interface{}{
					"type":      "response_networks",
					"agentId":   agentID,
					"requester": requester,
					"data":      getNetworksDetails(),
				})

			} else if mType == "request_wifi" {

				c.WriteJSON(map[string]interface{}{
					"type":      "response_wifi",
					"agentId":   agentID,
					"requester": requester,
					"data":      getWifiDetails(),
				})

			} else if mType == "request_seaweed" {

				c.WriteJSON(map[string]interface{}{
					"type":      "response_seaweed",
					"agentId":   agentID,
					"requester": requester,
					"data":      getSeaweedfsDetails(),
				})

			} else if mType == "restart_seaweed" {

				unit, _ := msg["unit"].(string)

				exec.Command("systemctl", "restart", unit).Run()

				c.WriteJSON(map[string]interface{}{
					"type":      "response_seaweed",
					"agentId":   agentID,
					"requester": requester,
					"data":      getSeaweedfsDetails(),
				})

			} else if mType == "request_seaweed_logs" {

				unit, _ := msg["unit"].(string)

				var logEntries []map[string]interface{}

				if runtime.GOOS != "windows" {

					cmd := exec.Command(
						"journalctl",
						"-u",
						unit,
						"-n",
						"100",
						"-o",
						"json",
						"--no-pager",
					)

					out, err := cmd.CombinedOutput()

					if err == nil {

						lines := strings.Split(
							strings.TrimSpace(string(out)),
							"\n",
						)

						for _, line := range lines {

							if line == "" {
								continue
							}

							var entry map[string]interface{}

							if parseErr := json.Unmarshal(
								[]byte(line),
								&entry,
							); parseErr == nil {

								ts := ""

								if rt, ok := entry["__REALTIME_TIMESTAMP"].(string); ok {
									ts = rt
								}

								msg := ""

								if m, ok := entry["MESSAGE"].(string); ok {
									msg = m
								}

								priority := "6"

								if p, ok := entry["PRIORITY"].(string); ok {
									priority = p
								}

								logEntries = append(
									logEntries,
									map[string]interface{}{
										"ts":  ts,
										"msg": msg,
										"p":   priority,
									},
								)
							}
						}
					}
				}

				if logEntries == nil {
					logEntries = []map[string]interface{}{}
				}

				c.WriteJSON(map[string]interface{}{
					"type":      "response_seaweed_logs",
					"agentId":   agentID,
					"requester": requester,
					"data":      logEntries,
				})

			} else if mType == "open_workrave" {

				err := OpenWorkrave()

				if err != nil {

					log.Println("Error abriendo Workrave:", err)

					c.WriteJSON(map[string]interface{}{
						"type":    "workrave_status",
						"agentId": agentID,
						"running": false,
					})

				} else {

					log.Println("Orden recibida: ABRIR WORKRAVE")

					time.Sleep(500 * time.Millisecond)

					running := isWorkraveRunning()

					log.Println(
						"Estado Workrave después de abrir:",
						running,
					)

					c.WriteJSON(map[string]interface{}{
						"type":    "workrave_status",
						"agentId": agentID,
						"running": running,
					})
				}

			} else if mType == "close_workrave" {

				err := CloseWorkrave()

				if err != nil {

					log.Println("Error cerrando Workrave:", err)

					c.WriteJSON(map[string]interface{}{
						"type":    "workrave_status",
						"agentId": agentID,
						"running": true,
					})

				} else {

					log.Println("Orden recibida: CERRAR WORKRAVE")

					time.Sleep(500 * time.Millisecond)

					running := isWorkraveRunning()

					log.Println(
						"Estado Workrave después de cerrar:",
						running,
					)

					c.WriteJSON(map[string]interface{}{
						"type":    "workrave_status",
						"agentId": agentID,
						"running": running,
					})
				}

			} else if mType == "run_command" {

				cmdStr, _ := msg["command"].(string)

				var b []byte

				if runtime.GOOS == "windows" {
					b, _ = exec.Command(
						"cmd",
						"/C",
						cmdStr,
					).CombinedOutput()
				} else {
					b, _ = exec.Command(
						"sh",
						"-c",
						cmdStr,
					).CombinedOutput()
				}

				c.WriteJSON(map[string]interface{}{
					"type":      "response_command",
					"agentId":   agentID,
					"requester": requester,
					"output":    string(b),
				})
			}
		}
	}
}

func startHTTPServer() {
	http.HandleFunc("/screen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		img := getRawScreenshot()
		if img == nil {
			http.Error(w, "Failed to capture", 500)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		jpeg.Encode(w, img, &jpeg.Options{Quality: 60})
	})
	log.Println("Starting local screen server on :8081 (Direct Tailscale Access)")
	http.ListenAndServe(":8081", nil)
}

func getRawScreenshot() *image.RGBA {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil
	}
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil
	}
	return img
}

func main() {

	log.Printf("3CX activo: %v", is3CXRunning())

	loadConfig()

	if path, err := getWorkravePath(); err != nil {

		log.Println(
			"Advertencia: no se pudo localizar Workrave:",
			err,
		)

	} else {

		log.Println(
			"Workrave detectado:",
			path,
		)
	}

	go MonitorTodayStats()

	agentInfo = GetAgentInfo()

	go startDashboard()

	go monitorMicroBreak()

	go monitor3CX()

	go monitorWorkrave()

	go startHTTPServer()

	go monitorAgentRegistration()

	go enviarHeartbeat()

	connectWS()

	select {}

	// go sendTelemetry()
}
