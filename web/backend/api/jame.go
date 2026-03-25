package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
)

// registerJameRoutes binds Jame Channel management endpoints to the ServeMux.
func (h *Handler) registerJameRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/jame/token", h.handleGetJameToken)
	mux.HandleFunc("POST /api/jame/token", h.handleRegenJameToken)
	mux.HandleFunc("POST /api/jame/setup", h.handleJameSetup)

	// WebSocket proxy: forward /jame/ws to gateway
	// This allows the frontend to connect via the same port as the web UI,
	// avoiding the need to expose extra ports for WebSocket communication.
	mux.HandleFunc("GET /jame/ws", h.handleWebSocketProxy())
}

// createWsProxy creates a reverse proxy to the current gateway WebSocket endpoint.
// The gateway bind host and port are resolved from the latest configuration.
func (h *Handler) createWsProxy() *httputil.ReverseProxy {
	wsProxy := httputil.NewSingleHostReverseProxy(h.gatewayProxyURL())
	wsProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
	}
	return wsProxy
}

// handleWebSocketProxy wraps a reverse proxy to handle WebSocket connections.
// The reverse proxy forwards the incoming upgrade handshake as-is.
func (h *Handler) handleWebSocketProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy := h.createWsProxy()
		proxy.ServeHTTP(w, r)
	}
}

// handleGetJameToken returns the current WS token and URL for the frontend.
//
//	GET /api/jame/token
func (h *Handler) handleGetJameToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	wsURL := h.buildWsURL(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":   cfg.Channels.Jame.Token(),
		"ws_url":  wsURL,
		"enabled": cfg.Channels.Jame.Enabled,
	})
}

// handleRegenJameToken generates a new Jame WebSocket token and saves it.
//
//	POST /api/jame/token
func (h *Handler) handleRegenJameToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	token := generateSecureToken()
	cfg.Channels.Jame.SetToken(token)

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	wsURL := h.buildWsURL(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":  token,
		"ws_url": wsURL,
	})
}

// ensureJameChannel enables the Jame channel with sane defaults if it isn't
// already configured. Returns true when the config was modified.
//
// callerOrigin is the Origin header from the setup request. If non-empty and
// no origins are configured yet, it's written as the allowed origin so the
// WebSocket handshake works for whatever host the caller is on (LAN, custom
// port, etc.). Pass "" when there's no request context.
func (h *Handler) ensureJameChannel(callerOrigin string) (bool, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %w", err)
	}

	changed := false

	if !cfg.Channels.Jame.Enabled {
		cfg.Channels.Jame.Enabled = true
		changed = true
	}

	if cfg.Channels.Jame.Token() == "" {
		cfg.Channels.Jame.SetToken(generateSecureToken())
		changed = true
	}

	// Seed origins from the request instead of hardcoding ports.
	if len(cfg.Channels.Jame.AllowOrigins) == 0 && callerOrigin != "" {
		cfg.Channels.Jame.AllowOrigins = []string{callerOrigin}
		changed = true
	}

	if changed {
		if err := config.SaveConfig(h.configPath, cfg); err != nil {
			return false, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return changed, nil
}

// handleJameSetup automatically configures everything needed for the Jame Channel to work.
//
//	POST /api/jame/setup
func (h *Handler) handleJameSetup(w http.ResponseWriter, r *http.Request) {
	changed, err := h.ensureJameChannel(r.Header.Get("Origin"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	wsURL := h.buildWsURL(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":   cfg.Channels.Jame.Token(),
		"ws_url":  wsURL,
		"enabled": true,
		"changed": changed,
	})
}

// generateSecureToken creates a random 32-character hex string.
func generateSecureToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to something pseudo-random if crypto/rand fails
		return fmt.Sprintf("jame_%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
