package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/extensions"
)

func (h *Handler) registerExtensionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/extensions/catalog", h.handleExtensionCatalog)
}

func (h *Handler) handleExtensionCatalog(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(extensions.Catalog{
		Providers: extensions.ProviderCatalog(cfg),
		Tools:     toolCatalogItems(),
		Channels:  channelCatalogItems(),
	})
}
