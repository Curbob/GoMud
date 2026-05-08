package web

import (
	"encoding/json"
	"net/http"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/observability"
)

func observabilityIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(
		configs.GetFilePathsConfig().AdminHtml.String()+"/_header.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/observability/index.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/_footer.html",
	)
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
	}

	data := struct {
		Events  []observability.RecentEvent
		Players []observability.OnlinePlayerSnapshot
	}{
		Events:  observability.RecentEvents(),
		Players: observability.OnlinePlayersSnapshot(),
	}

	if err := tmpl.Execute(w, data); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}
}

func observabilityEventsAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, observability.RecentEvents())
}

func observabilityPlayersAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, observability.OnlinePlayersSnapshot())
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		mudlog.Error("HTTP JSON", "error", err)
		http.Error(w, "failed to encode json", http.StatusInternalServerError)
	}
}
