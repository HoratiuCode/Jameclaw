package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/extensions"
)

type toolCatalogEntry struct {
	Name        string
	Description string
	Category    string
	ConfigKey   string
}

type toolSupportItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	ConfigKey   string `json:"config_key"`
	Status      string `json:"status"`
	ReasonCode  string `json:"reason_code,omitempty"`
}

type toolSupportResponse struct {
	Tools []toolSupportItem `json:"tools"`
}

type toolStateRequest struct {
	Enabled bool `json:"enabled"`
}

type mcpServerResponse struct {
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Transport string   `json:"transport"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	URL       string   `json:"url,omitempty"`
}

type mcpServerRequest struct {
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	URL       string   `json:"url"`
}

var toolCatalog = []toolCatalogEntry{
	{
		Name:        "read_file",
		Description: "Read file content from the workspace or explicitly allowed paths.",
		Category:    "filesystem",
		ConfigKey:   "read_file",
	},
	{
		Name:        "write_file",
		Description: "Create or overwrite files within the writable workspace scope.",
		Category:    "filesystem",
		ConfigKey:   "write_file",
	},
	{
		Name:        "list_dir",
		Description: "Inspect directories and enumerate files available to the agent.",
		Category:    "filesystem",
		ConfigKey:   "list_dir",
	},
	{
		Name:        "edit_file",
		Description: "Apply targeted edits to existing files without rewriting everything.",
		Category:    "filesystem",
		ConfigKey:   "edit_file",
	},
	{
		Name:        "append_file",
		Description: "Append content to the end of an existing file.",
		Category:    "filesystem",
		ConfigKey:   "append_file",
	},
	{
		Name:        "exec",
		Description: "Run shell commands inside the configured workspace sandbox.",
		Category:    "filesystem",
		ConfigKey:   "exec",
	},
	{
		Name:        "mac_control",
		Description: "Open macOS apps, URLs, local paths, and browser search results.",
		Category:    "automation",
		ConfigKey:   "mac_control",
	},
	{
		Name:        "cron",
		Description: "Schedule one-time or recurring reminders, jobs, and shell commands.",
		Category:    "automation",
		ConfigKey:   "cron",
	},
	{
		Name:        "web_search",
		Description: "Search the web using the configured providers.",
		Category:    "web",
		ConfigKey:   "web",
	},
	{
		Name:        "web_fetch",
		Description: "Fetch and summarize the contents of a webpage.",
		Category:    "web",
		ConfigKey:   "web_fetch",
	},
	{
		Name:        "message",
		Description: "Send a follow-up message back to the active user or chat.",
		Category:    "communication",
		ConfigKey:   "message",
	},
	{
		Name:        "send_file",
		Description: "Send an outbound file or media attachment to the active chat.",
		Category:    "communication",
		ConfigKey:   "send_file",
	},
	{
		Name:        "screenshot",
		Description: "Capture the visible desktop and send it to the active chat when requested.",
		Category:    "communication",
		ConfigKey:   "screenshot",
	},
	{
		Name:        "find_skills",
		Description: "Search external skill registries for installable skills.",
		Category:    "skills",
		ConfigKey:   "find_skills",
	},
	{
		Name:        "install_skill",
		Description: "Install a skill into the current workspace from a registry.",
		Category:    "skills",
		ConfigKey:   "install_skill",
	},
	{
		Name:        "spawn",
		Description: "Launch a background subagent for long-running or delegated work.",
		Category:    "agents",
		ConfigKey:   "spawn",
	},
	{
		Name:        "spawn_status",
		Description: "Query the status of spawned subagents.",
		Category:    "agents",
		ConfigKey:   "spawn_status",
	},
	{
		Name:        "i2c",
		Description: "Interact with I2C hardware devices exposed on the host.",
		Category:    "hardware",
		ConfigKey:   "i2c",
	},
	{
		Name:        "spi",
		Description: "Interact with SPI hardware devices exposed on the host.",
		Category:    "hardware",
		ConfigKey:   "spi",
	},
	{
		Name:        "tool_search_tool_regex",
		Description: "Discover hidden MCP tools by regex search when tool discovery is enabled.",
		Category:    "discovery",
		ConfigKey:   "mcp.discovery.use_regex",
	},
	{
		Name:        "tool_search_tool_bm25",
		Description: "Discover hidden MCP tools by semantic ranking when tool discovery is enabled.",
		Category:    "discovery",
		ConfigKey:   "mcp.discovery.use_bm25",
	},
}

func toolCatalogItems() []extensions.CatalogItem {
	items := make([]extensions.CatalogItem, 0, len(toolCatalog))
	for _, entry := range toolCatalog {
		items = append(items, extensions.CatalogItem{
			ID:          entry.Name,
			Name:        entry.Name,
			Category:    entry.Category,
			Description: entry.Description,
			ConfigKey:   entry.ConfigKey,
		})
	}
	return items
}

func (h *Handler) registerToolRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tools", h.handleListTools)
	mux.HandleFunc("PUT /api/tools/{name}/state", h.handleUpdateToolState)
	mux.HandleFunc("GET /api/tools/mcp/servers", h.handleListMCPServers)
	mux.HandleFunc("POST /api/tools/mcp/servers", h.handleSaveMCPServer)
}

func (h *Handler) handleListMCPServers(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	servers := make([]mcpServerResponse, 0, len(cfg.Tools.MCP.Servers))
	for name, server := range cfg.Tools.MCP.Servers {
		transport := strings.TrimSpace(server.Type)
		if transport == "" {
			if strings.TrimSpace(server.URL) != "" {
				transport = "http"
			} else {
				transport = "stdio"
			}
		}
		servers = append(servers, mcpServerResponse{
			Name: name, Enabled: server.Enabled, Transport: transport,
			Command: server.Command, Args: server.Args, URL: server.URL,
		})
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"enabled": cfg.Tools.MCP.Enabled, "servers": servers})
}

func (h *Handler) handleSaveMCPServer(w http.ResponseWriter, r *http.Request) {
	var req mcpServerRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	req.Name = strings.TrimSpace(req.Name)
	req.Transport = strings.ToLower(strings.TrimSpace(req.Transport))
	req.Command = strings.TrimSpace(req.Command)
	req.URL = strings.TrimSpace(req.URL)
	if req.Name == "" {
		http.Error(w, "server name is required", http.StatusBadRequest)
		return
	}
	if req.Transport != "stdio" && req.Transport != "http" && req.Transport != "sse" {
		http.Error(w, "transport must be stdio, http, or sse", http.StatusBadRequest)
		return
	}
	if req.Transport == "stdio" && req.Command == "" {
		http.Error(w, "command is required for a CLI MCP server", http.StatusBadRequest)
		return
	}
	if req.Transport != "stdio" && req.URL == "" {
		http.Error(w, "URL is required for an HTTP or SSE MCP server", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	if cfg.Tools.MCP.Servers == nil {
		cfg.Tools.MCP.Servers = make(map[string]config.MCPServerConfig)
	}
	cfg.Tools.MCP.Enabled = true
	cfg.Tools.MCP.Servers[req.Name] = config.MCPServerConfig{
		Enabled: req.Enabled, Type: req.Transport, Command: req.Command,
		Args: req.Args, URL: req.URL,
	}
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	restarted := false
	if _, err := h.RestartGateway(); err == nil {
		restarted = true
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "gateway_restarted": restarted})
}

func (h *Handler) handleListTools(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolSupportResponse{
		Tools: buildToolSupport(cfg),
	})
}

func (h *Handler) handleUpdateToolState(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	var req toolStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := applyToolState(cfg, r.PathValue("name"), req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func buildToolSupport(cfg *config.Config) []toolSupportItem {
	items := make([]toolSupportItem, 0, len(toolCatalog))
	for _, entry := range toolCatalog {
		status := "disabled"
		reasonCode := ""

		switch entry.Name {
		case "find_skills", "install_skill":
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				if cfg.Tools.IsToolEnabled("skills") {
					status = "enabled"
				} else {
					status = "blocked"
					reasonCode = "requires_skills"
				}
			}
		case "spawn", "spawn_status":
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				if cfg.Tools.IsToolEnabled("subagent") {
					status = "enabled"
				} else {
					status = "blocked"
					reasonCode = "requires_subagent"
				}
			}
		case "tool_search_tool_regex":
			status, reasonCode = resolveDiscoveryToolSupport(cfg, cfg.Tools.MCP.Discovery.UseRegex)
		case "tool_search_tool_bm25":
			status, reasonCode = resolveDiscoveryToolSupport(cfg, cfg.Tools.MCP.Discovery.UseBM25)
		case "i2c", "spi":
			status, reasonCode = resolveHardwareToolSupport(cfg.Tools.IsToolEnabled(entry.ConfigKey))
		case "mac_control":
			status, reasonCode = resolveMacControlToolSupport(cfg.Tools.IsToolEnabled(entry.ConfigKey))
		default:
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				status = "enabled"
			}
		}

		items = append(items, toolSupportItem{
			Name:        entry.Name,
			Description: entry.Description,
			Category:    entry.Category,
			ConfigKey:   entry.ConfigKey,
			Status:      status,
			ReasonCode:  reasonCode,
		})
	}
	return items
}

func resolveMacControlToolSupport(enabled bool) (string, string) {
	if !enabled {
		return "disabled", ""
	}
	if runtime.GOOS != "darwin" {
		return "blocked", "requires_macos"
	}
	return "enabled", ""
}

func resolveHardwareToolSupport(enabled bool) (string, string) {
	if !enabled {
		return "disabled", ""
	}
	if runtime.GOOS != "linux" {
		return "blocked", "requires_linux"
	}
	return "enabled", ""
}

func resolveDiscoveryToolSupport(cfg *config.Config, methodEnabled bool) (string, string) {
	if !cfg.Tools.IsToolEnabled("mcp") {
		return "disabled", ""
	}
	if !cfg.Tools.MCP.Discovery.Enabled {
		return "blocked", "requires_mcp_discovery"
	}
	if !methodEnabled {
		return "disabled", ""
	}
	return "enabled", ""
}

func applyToolState(cfg *config.Config, toolName string, enabled bool) error {
	switch toolName {
	case "read_file":
		cfg.Tools.ReadFile.Enabled = enabled
	case "write_file":
		cfg.Tools.WriteFile.Enabled = enabled
	case "list_dir":
		cfg.Tools.ListDir.Enabled = enabled
	case "edit_file":
		cfg.Tools.EditFile.Enabled = enabled
	case "append_file":
		cfg.Tools.AppendFile.Enabled = enabled
	case "exec":
		cfg.Tools.Exec.Enabled = enabled
	case "mac_control":
		cfg.Tools.MacControl.Enabled = enabled
	case "cron":
		cfg.Tools.Cron.Enabled = enabled
	case "web_search":
		cfg.Tools.Web.Enabled = enabled
	case "web_fetch":
		cfg.Tools.WebFetch.Enabled = enabled
	case "message":
		cfg.Tools.Message.Enabled = enabled
	case "send_file":
		cfg.Tools.SendFile.Enabled = enabled
	case "screenshot":
		cfg.Tools.Screenshot.Enabled = enabled
	case "find_skills":
		cfg.Tools.FindSkills.Enabled = enabled
		if enabled {
			cfg.Tools.Skills.Enabled = true
		}
	case "install_skill":
		cfg.Tools.InstallSkill.Enabled = enabled
		if enabled {
			cfg.Tools.Skills.Enabled = true
		}
	case "spawn":
		cfg.Tools.Spawn.Enabled = enabled
		if enabled {
			cfg.Tools.Subagent.Enabled = true
		}
	case "spawn_status":
		cfg.Tools.SpawnStatus.Enabled = enabled
		if enabled {
			cfg.Tools.Spawn.Enabled = true
			cfg.Tools.Subagent.Enabled = true
		}
	case "i2c":
		cfg.Tools.I2C.Enabled = enabled
	case "spi":
		cfg.Tools.SPI.Enabled = enabled
	case "tool_search_tool_regex":
		cfg.Tools.MCP.Discovery.UseRegex = enabled
		if enabled {
			cfg.Tools.MCP.Enabled = true
			cfg.Tools.MCP.Discovery.Enabled = true
		}
	case "tool_search_tool_bm25":
		cfg.Tools.MCP.Discovery.UseBM25 = enabled
		if enabled {
			cfg.Tools.MCP.Enabled = true
			cfg.Tools.MCP.Discovery.Enabled = true
		}
	default:
		return fmt.Errorf("tool %q cannot be updated", toolName)
	}
	return nil
}
