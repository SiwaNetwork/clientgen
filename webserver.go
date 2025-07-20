package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config represents the clientgen configuration
type Config struct {
	Iface                             string  `json:"Iface"`
	ServerMAC                         string  `json:"ServerMAC"`
	ServerAddress                     string  `json:"ServerAddress"`
	ClientIPStart                     string  `json:"ClientIPStart"`
	ClientIPEnd                       string  `json:"ClientIPEnd"`
	ClientIPStep                      uint    `json:"ClientIPStep"`
	SoftStartRate                     uint64  `json:"SoftStartRate"`
	TimeoutSec                        float64 `json:"TimeoutSec"`
	DurationSec                       float64 `json:"DurationSec"`
	TimeAfterDurationBeforeRestartSec float64 `json:"TimeAfterDurationBeforeRestartSec"`
	TimeBetweenDelayReqSec            float64 `json:"TimeBetweenDelayReqSec"`
	ClientRetranTimeWhenNoResponseSec float64 `json:"ClientRetranTimeWhenNoResponseSec"`
	NumTXWorkers                      int     `json:"NumTXWorkers"`
	NumTXTSWorkerPerTx                int     `json:"NumTXTSWorkerPerTx"`
	NumRXWorkers                      int     `json:"NumRXWorkers"`
	NumPacketParsers                  int     `json:"NumPacketParsers"`
	NumPacketProcessors               int     `json:"NumPacketProcessors"`
	NumClientRetransmitProcs          int     `json:"NumClientRetransmitProcs"`
	NumClientRestartProcs             int     `json:"NumClientRestartProcs"`
	RestartClientsAfterDuration       bool    `json:"RestartClientsAfterDuration"`
	DebugPrint                        bool    `json:"DebugPrint"`
	DebugLogClient                    bool    `json:"DebugLogClient"`
	DebugIoWkrRX                      bool    `json:"DebugIoWkrRX"`
	DebugIoWkrTX                      bool    `json:"DebugIoWkrTX"`
	DebugDetailPerf                   bool    `json:"DebugDetailPerf"`
	DebugRestartProc                  bool    `json:"DebugRestartProc"`
	DebugRetransProc                  bool    `json:"DebugRetransProc"`
	DebugProfilers                    bool    `json:"DebugProfilers"`
	PrintPerformance                  bool    `json:"PrintPerformance"`
	PrintClientData                   bool    `json:"PrintClientData"`
	PrintTxRxCounts                   bool    `json:"PrintTxRxCounts"`
	PrintClientReqData                bool    `json:"PrintClientReqData"`
	PrintLatencyData                  bool    `json:"PrintLatencyData"`
	CounterPrintIntervalSecs          uint    `json:"CounterPrintIntervalSecs"`
}

// Statistics represents runtime statistics
type Statistics struct {
	TotalClients                    uint64 `json:"TotalClients"`
	TotalPacketsSent                uint64 `json:"TotalPacketsSent"`
	TotalPacketsRcvd                uint64 `json:"TotalPacketsRcvd"`
	TotalTXTSPacketsSent            uint64 `json:"TotalTXTSPacketsSent"`
	TotalTXTSRead                   uint64 `json:"TotalTXTSRead"`
	MaxTXTSBytesOutstanding         uint64 `json:"MaxTXTSBytesOutstanding"`
	TotalGenMsgSent                 uint64 `json:"TotalGenMsgSent"`
	TotalGenMsgRcvd                 uint64 `json:"TotalGenMsgRcvd"`
	TotalEventMsgSent               uint64 `json:"TotalEventMsgSent"`
	TotalEventMsgRcvd               uint64 `json:"TotalEventMsgRcvd"`
	TotalClientAnnounceReq          uint64 `json:"TotalClientAnnounceReq"`
	TotalClientAnnounceReqResend    uint64 `json:"TotalClientAnnounceReqResend"`
	TotalClientAnnounceGrant        uint64 `json:"TotalClientAnnounceGrant"`
	TotalClientSyncReq              uint64 `json:"TotalClientSyncReq"`
	TotalClientSyncReqResend        uint64 `json:"TotalClientSyncReqResend"`
	TotalClientSyncGrant            uint64 `json:"TotalClientSyncGrant"`
	TotalClientDelayRespReq         uint64 `json:"TotalClientDelayRespReq"`
	TotalClientDelayRespReqResend   uint64 `json:"TotalClientDelayRespReqResend"`
	TotalClientDelayRespGrant       uint64 `json:"TotalClientDelayRespGrant"`
	TotalSyncRcvd                   uint64 `json:"TotalSyncRcvd"`
	TotalPDelayRespRcvd             uint64 `json:"TotalPDelayRespRcvd"`
	TotalFollowUpRcvd               uint64 `json:"TotalFollowUpRcvd"`
	TotalPDelayRespFollowUpRcvd     uint64 `json:"TotalPDelayRespFollowUpRcvd"`
	TotalAnnounceRcvd               uint64 `json:"TotalAnnounceRcvd"`
	TotalDelayReqSent               uint64 `json:"TotalDelayReqSent"`
	TotalDelayRespRcvd              uint64 `json:"TotalDelayRespRcvd"`
	PFRingRXPackets                 uint64 `json:"PFRingRXPackets"`
	PFRingRXBytes                   uint64 `json:"PFRingRXBytes"`
	PFRingRXDropped                 uint64 `json:"PFRingRXDropped"`
	PFRingTXPackets                 uint64 `json:"PFRingTXPackets"`
	PFRingTXBytes                   uint64 `json:"PFRingTXBytes"`
	PFRingHWTimestamps              uint64 `json:"PFRingHWTimestamps"`
}

// WebServer struct to manage the web interface
type WebServer struct {
	config      *Config
	configMutex sync.RWMutex
	stats       Statistics
	statsMutex  sync.RWMutex
	isRunning   bool
	runMutex    sync.RWMutex
	cancelFunc  context.CancelFunc
	port        string
}

// NewWebServer creates a new web server instance
func NewWebServer(port string) *WebServer {
	return &WebServer{
		port:      port,
		isRunning: false,
		stats:     Statistics{},
	}
}

// Start starts the web server
func (ws *WebServer) Start() error {
	// Load default configuration
	ws.loadDefaultConfig()

	// Setup HTTP routes
	http.HandleFunc("/", ws.handleIndex)
	http.HandleFunc("/api/config", ws.handleConfig)
	http.HandleFunc("/api/stats", ws.handleStats)
	http.HandleFunc("/api/stats/clear", ws.handleStatsClear)
	http.HandleFunc("/api/start", ws.handleStart)
	http.HandleFunc("/api/stop", ws.handleStop)
	http.HandleFunc("/api/status", ws.handleStatus)

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	log.Printf("Веб-сервер запущен на порту %s", ws.port)
	log.Printf("Откройте http://localhost%s в браузере", ws.port)
	
	return http.ListenAndServe(ws.port, nil)
}

// Load default configuration
func (ws *WebServer) loadDefaultConfig() {
	ws.configMutex.Lock()
	defer ws.configMutex.Unlock()

	// Try to load existing config file
	if _, err := os.Stat("clientgen_config.json"); err == nil {
		file, err := os.Open("clientgen_config.json")
		if err == nil {
			defer file.Close()
			decoder := json.NewDecoder(file)
			config := &Config{}
			if decoder.Decode(config) == nil {
				ws.config = config
				return
			}
		}
	}

	// Default configuration if file doesn't exist or failed to load
	ws.config = &Config{
		Iface:                             "ens1f0np0",
		ServerMAC:                         "0c:42:a1:80:31:66",
		ServerAddress:                     "2401:db00:eef0:1120:3520:0:1401:eb11",
		ClientIPStart:                     "2401:db00:eef0:1120:3520:0:1401:eb14",
		ClientIPEnd:                       "2401:db00:eef0:1120:3520:0:1403:e6e4",
		ClientIPStep:                      1,
		SoftStartRate:                     1000000000000000,
		TimeoutSec:                        90,
		DurationSec:                       5,
		TimeAfterDurationBeforeRestartSec: 1,
		TimeBetweenDelayReqSec:            1,
		ClientRetranTimeWhenNoResponseSec: 1,
		NumTXWorkers:                      4,
		NumTXTSWorkerPerTx:                3,
		NumRXWorkers:                      4,
		NumPacketParsers:                  4,
		NumPacketProcessors:               4,
		NumClientRetransmitProcs:          4,
		NumClientRestartProcs:             4,
		RestartClientsAfterDuration:       true,
		DebugPrint:                        false,
		DebugLogClient:                    false,
		DebugIoWkrRX:                      false,
		DebugIoWkrTX:                      false,
		DebugDetailPerf:                   false,
		DebugRestartProc:                  false,
		DebugRetransProc:                  false,
		DebugProfilers:                    false,
		PrintPerformance:                  true,
		PrintClientData:                   true,
		PrintTxRxCounts:                   true,
		PrintClientReqData:                false,
		PrintLatencyData:                  true,
		CounterPrintIntervalSecs:          1,
	}
}

// Handle main page
func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	indexPath := filepath.Join("web", "templates", "index.html")
	http.ServeFile(w, r, indexPath)
}

// Handle configuration API
func (ws *WebServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		ws.configMutex.RLock()
		config := ws.config
		ws.configMutex.RUnlock()

		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, "Failed to encode configuration", http.StatusInternalServerError)
			return
		}

	case "POST":
		ws.configMutex.Lock()
		defer ws.configMutex.Unlock()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var newConfig Config
		if err := json.Unmarshal(body, &newConfig); err != nil {
			http.Error(w, "Failed to parse configuration", http.StatusBadRequest)
			return
		}

		ws.config = &newConfig

		// Save configuration to file
		if err := ws.saveConfigToFile(); err != nil {
			http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handle statistics API
func (ws *WebServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.statsMutex.RLock()
	stats := ws.stats
	ws.statsMutex.RUnlock()

	// Simulate some dynamic statistics for demo purposes
	ws.statsMutex.Lock()
	if ws.isRunning {
		ws.stats.TotalPacketsSent += 100
		ws.stats.TotalPacketsRcvd += 95
		ws.stats.PFRingRXPackets += 95
		ws.stats.PFRingTXPackets += 100
		ws.stats.TotalSyncRcvd += 10
		ws.stats.TotalAnnounceRcvd += 5
		ws.stats.TotalDelayReqSent += 20
		ws.stats.PFRingHWTimestamps += 50
	}
	stats = ws.stats
	ws.statsMutex.Unlock()

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode statistics", http.StatusInternalServerError)
	}
}

// Handle clear statistics API
func (ws *WebServer) handleStatsClear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.statsMutex.Lock()
	ws.stats = Statistics{}
	ws.statsMutex.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Handle start API
func (ws *WebServer) handleStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.runMutex.Lock()
	defer ws.runMutex.Unlock()

	if ws.isRunning {
		http.Error(w, "System is already running", http.StatusBadRequest)
		return
	}

	ws.configMutex.RLock()
	config := ws.config
	ws.configMutex.RUnlock()

	if config == nil {
		http.Error(w, "No configuration available", http.StatusBadRequest)
		return
	}

	// Create context for simulation
	ctx, cancel := context.WithCancel(context.Background())
	ws.cancelFunc = cancel

	// Start simulation in a goroutine (for demo purposes)
	go func() {
		defer func() {
			ws.runMutex.Lock()
			ws.isRunning = false
			ws.runMutex.Unlock()
		}()

		log.Println("Simulating ClientGen start...")
		
		// Initialize some baseline statistics
		ws.statsMutex.Lock()
		ws.stats.TotalClients = 1000
		ws.statsMutex.Unlock()

		// Wait for cancellation or timeout
		select {
		case <-ctx.Done():
			log.Println("ClientGen simulation stopped")
		case <-time.After(time.Duration(config.TimeoutSec) * time.Second):
			log.Println("ClientGen simulation timed out")
		}
	}()

	ws.isRunning = true

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// Handle stop API
func (ws *WebServer) handleStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.runMutex.Lock()
	defer ws.runMutex.Unlock()

	if !ws.isRunning {
		http.Error(w, "System is not running", http.StatusBadRequest)
		return
	}

	if ws.cancelFunc != nil {
		ws.cancelFunc()
		ws.cancelFunc = nil
	}

	ws.isRunning = false

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

// Handle status API
func (ws *WebServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.runMutex.RLock()
	isRunning := ws.isRunning
	ws.runMutex.RUnlock()

	status := map[string]interface{}{
		"running": isRunning,
		"uptime":  time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(status)
}

// Save configuration to file
func (ws *WebServer) saveConfigToFile() error {
	file, err := os.Create("clientgen_config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	return encoder.Encode(ws.config)
}

// Main function for the web server
func main() {
	port := ":8080"
	if len(os.Args) > 1 {
		port = ":" + os.Args[1]
	}

	webServer := NewWebServer(port)
	
	// Create web directories if they don't exist
	os.MkdirAll("web/static/css", 0755)
	os.MkdirAll("web/static/js", 0755)
	os.MkdirAll("web/templates", 0755)

	if err := webServer.Start(); err != nil {
		log.Fatal("Failed to start web server:", err)
	}
}