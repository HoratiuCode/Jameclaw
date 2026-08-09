package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
)

type researchProviderStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RequiresKey bool   `json:"requires_api_key"`
	Configured  bool   `json:"configured"`
	Enabled     bool   `json:"enabled"`
}

func (h *Handler) registerResearchProviderRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/research-providers", h.handleListResearchProviders)
	mux.HandleFunc("POST /api/research-providers/{id}", h.handleConnectResearchProvider)
}

func (h *Handler) handleListResearchProviders(w http.ResponseWriter, _ *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"providers":       researchProviderStatuses(cfg),
		"active_provider": activeResearchProvider(cfg),
	})
}

func (h *Handler) handleConnectResearchProvider(w http.ResponseWriter, r *http.Request) {
	providerID := strings.TrimSpace(r.PathValue("id"))
	var request struct {
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "Invalid research provider request.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	request.APIKey = strings.TrimSpace(request.APIKey)

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	if request.Enabled {
		disableResearchProviders(cfg)
		cfg.Tools.Web.Enabled = true
		// A dedicated research connection should be used instead of silently
		// preferring the active chat model's native search implementation.
		cfg.Tools.Web.PreferNative = false
	}

	switch providerID {
	case "brave":
		if request.Enabled && request.APIKey == "" && cfg.Tools.Web.Brave.APIKey() == "" {
			http.Error(w, "A Brave Search API key is required.", http.StatusUnprocessableEntity)
			return
		}
		if request.APIKey != "" {
			cfg.Tools.Web.Brave.SetAPIKey(request.APIKey)
		}
		cfg.Tools.Web.Brave.Enabled = request.Enabled
	case "tavily":
		if request.Enabled && request.APIKey == "" && cfg.Tools.Web.Tavily.APIKey() == "" {
			http.Error(w, "A Tavily API key is required.", http.StatusUnprocessableEntity)
			return
		}
		if request.APIKey != "" {
			cfg.Tools.Web.Tavily.SetAPIKey(request.APIKey)
		}
		cfg.Tools.Web.Tavily.Enabled = request.Enabled
	case "perplexity":
		if request.Enabled && request.APIKey == "" && cfg.Tools.Web.Perplexity.APIKey() == "" {
			http.Error(w, "A Perplexity API key is required.", http.StatusUnprocessableEntity)
			return
		}
		if request.APIKey != "" {
			cfg.Tools.Web.Perplexity.SetAPIKey(request.APIKey)
		}
		cfg.Tools.Web.Perplexity.Enabled = request.Enabled
	case "duckduckgo":
		cfg.Tools.Web.DuckDuckGo.Enabled = request.Enabled
	default:
		http.Error(w, "Unsupported research provider.", http.StatusNotFound)
		return
	}

	if !request.Enabled {
		cfg.Tools.Web.Enabled = anyResearchProviderEnabled(cfg)
	}
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save research provider: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":                   "ok",
		"active_provider":          activeResearchProvider(cfg),
		"gateway_restart_required": true,
	})
}

func researchProviderStatuses(cfg *config.Config) []researchProviderStatus {
	return []researchProviderStatus{
		{ID: "tavily", Name: "Tavily", Description: "Research-focused search with concise source-backed results.", RequiresKey: true, Configured: cfg.Tools.Web.Tavily.APIKey() != "", Enabled: cfg.Tools.Web.Tavily.Enabled},
		{ID: "brave", Name: "Brave Search", Description: "Independent web search with broad index coverage.", RequiresKey: true, Configured: cfg.Tools.Web.Brave.APIKey() != "", Enabled: cfg.Tools.Web.Brave.Enabled},
		{ID: "perplexity", Name: "Perplexity", Description: "Answer-oriented web research and citations.", RequiresKey: true, Configured: cfg.Tools.Web.Perplexity.APIKey() != "", Enabled: cfg.Tools.Web.Perplexity.Enabled},
		{ID: "duckduckgo", Name: "DuckDuckGo", Description: "Keyless web research for a quick local setup.", RequiresKey: false, Configured: true, Enabled: cfg.Tools.Web.DuckDuckGo.Enabled},
	}
}

func disableResearchProviders(cfg *config.Config) {
	cfg.Tools.Web.Brave.Enabled = false
	cfg.Tools.Web.Tavily.Enabled = false
	cfg.Tools.Web.Perplexity.Enabled = false
	cfg.Tools.Web.DuckDuckGo.Enabled = false
}

func anyResearchProviderEnabled(cfg *config.Config) bool {
	return cfg.Tools.Web.Brave.Enabled || cfg.Tools.Web.Tavily.Enabled || cfg.Tools.Web.Perplexity.Enabled || cfg.Tools.Web.DuckDuckGo.Enabled
}

func activeResearchProvider(cfg *config.Config) string {
	switch {
	case cfg.Tools.Web.Tavily.Enabled:
		return "tavily"
	case cfg.Tools.Web.Brave.Enabled:
		return "brave"
	case cfg.Tools.Web.Perplexity.Enabled:
		return "perplexity"
	case cfg.Tools.Web.DuckDuckGo.Enabled:
		return "duckduckgo"
	default:
		return ""
	}
}
