package api

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/web/backend/launcherconfig"
)

// Handler serves HTTP API requests.
type Handler struct {
	configPath           string
	serverPort           int
	serverPublic         bool
	serverPublicExplicit bool
	serverCIDRs          []string
	oauthMu              sync.Mutex
	oauthFlows           map[string]*oauthFlow
	oauthState           map[string]string
	fileSearchMu         sync.Mutex
	fileSearchCache      fileSearchCache
	activeJameWebSockets atomic.Int64
}

// NewHandler creates an instance of the API handler.
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath: configPath,
		serverPort: launcherconfig.DefaultPort,
		oauthFlows: make(map[string]*oauthFlow),
		oauthState: make(map[string]string),
	}
}

// SetServerOptions stores current backend listen options for fallback behavior.
func (h *Handler) SetServerOptions(port int, public bool, publicExplicit bool, allowedCIDRs []string) {
	h.serverPort = port
	h.serverPublic = public
	h.serverPublicExplicit = publicExplicit
	h.serverCIDRs = append([]string(nil), allowedCIDRs...)
}

// RegisterRoutes binds all API endpoint handlers to the ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Config CRUD
	h.registerConfigRoutes(mux)

	// Jame Channel (WebSocket chat)
	h.registerJameRoutes(mux)

	// Gateway process lifecycle
	h.registerGatewayRoutes(mux)

	// Session history
	h.registerSessionRoutes(mux)
	h.registerUsageRoutes(mux)
	h.registerAgentRoutes(mux)
	h.registerDashboardRoutes(mux)
	h.registerAutomationRoutes(mux)
	h.registerAspirineRoutes(mux)
	h.registerFileRoutes(mux)

	// OAuth login and credential management
	h.registerOAuthRoutes(mux)

	// Model list management
	h.registerModelRoutes(mux)
	h.registerExtensionRoutes(mux)

	// Channel catalog (for frontend navigation/config pages)
	h.registerChannelRoutes(mux)

	// Skills and tools support/actions
	h.registerSkillRoutes(mux)
	h.registerToolRoutes(mux)

	// OS startup / launch-at-login
	h.registerStartupRoutes(mux)

	// Launcher service parameters (port/public)
	h.registerLauncherConfigRoutes(mux)

	// Sandboxed reverse proxy for local development previews. This makes a
	// localhost server started by the agent reachable from the same Web Console
	// session when the console itself is accessed over Tailscale.
	h.registerLocalPreviewRoutes(mux)

	// GitHub release update checks
	h.registerUpdateRoutes(mux)

	// First-run onboarding readiness
	h.registerOnboardingRoutes(mux)
}

// Shutdown gracefully shuts down the handler, stopping the gateway if it was started by this handler.
func (h *Handler) Shutdown() {
	h.StopGateway()
}

// WebActivityActive reports whether the local web console is actively connected
// to the Jame gateway through the launcher WebSocket proxy.
func (h *Handler) WebActivityActive() bool {
	return h.activeJameWebSockets.Load() > 0
}

// AgentActivityActive reports whether either the Web Console is connected or
// the gateway is currently running an agent turn from any channel.
func (h *Handler) AgentActivityActive() bool {
	if h.WebActivityActive() {
		return true
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false
	}
	healthResp, statusCode, err := h.getGatewayHealth(cfg, 500*time.Millisecond)
	return err == nil && statusCode == http.StatusOK && healthResp.AgentActive
}
