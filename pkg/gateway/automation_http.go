package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/cron"
)

// createAutomationRegistrar exposes a local, token-protected trigger used by
// the launcher. Jobs still execute inside the gateway, where the agent and
// cron handler are available.
func createAutomationRegistrar(cfg *config.Config, service *cron.CronService) func(*http.ServeMux) {
	return func(mux *http.ServeMux) {
		authorized := func(w http.ResponseWriter, r *http.Request) bool {
			token := ""
			if cfg != nil {
				token = strings.TrimSpace(cfg.Channels.Jame.Token())
			}
			if token == "" || r.Header.Get("Authorization") != "Bearer "+token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return false
			}
			return true
		}
		mux.HandleFunc("POST /automation/run/{id}", func(w http.ResponseWriter, r *http.Request) {
			if !authorized(w, r) {
				return
			}

			if err := service.RunNow(r.PathValue("id")); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "running"})
		})
		mux.HandleFunc("POST /automation/event/{event}", func(w http.ResponseWriter, r *http.Request) {
			if !authorized(w, r) {
				return
			}
			count := service.TriggerEvent(r.PathValue("event"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "matched_jobs": count})
		})
	}
}
