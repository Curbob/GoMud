package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/GoMudEngine/GoMud/mudbot/internal/leaderboard"
)

// HealthServer provides a simple HTTP health check endpoint
type HealthServer struct {
	dataProvider *leaderboard.DataProvider
	startTime    time.Time
	port         int
}

// NewHealthServer creates a new health check server
func NewHealthServer(dataProvider *leaderboard.DataProvider, port int) *HealthServer {
	return &HealthServer{
		dataProvider: dataProvider,
		startTime:    time.Now(),
		port:         port,
	}
}

// Start starts the health check server
func (hs *HealthServer) Start() error {
	http.HandleFunc("/health", hs.healthHandler)
	http.HandleFunc("/status", hs.statusHandler)

	fmt.Printf("Health server starting on port %d\n", hs.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", hs.port), nil)
}

// healthHandler provides simple health check
func (hs *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(hs.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// statusHandler provides detailed server status
func (hs *HealthServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serverStatus, err := hs.dataProvider.GetServerStatus()
	if err != nil {
		http.Error(w, "Failed to get server status", http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"bot": map[string]interface{}{
			"status":    "running",
			"uptime":    time.Since(hs.startTime).String(),
			"timestamp": time.Now().Format(time.RFC3339),
		},
		"server": serverStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}