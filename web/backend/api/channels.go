package api

import (
	"encoding/json"
	"net/http"

	"github.com/sipeed/jameclaw/pkg/extensions"
)

type channelCatalogItem struct {
	Name      string `json:"name"`
	ConfigKey string `json:"config_key"`
	Variant   string `json:"variant,omitempty"`
}

var channelCatalog = []channelCatalogItem{
	{Name: "telegram", ConfigKey: "telegram"},
	{Name: "whatsapp", ConfigKey: "whatsapp"},
	{Name: "whatsapp_native", ConfigKey: "whatsapp", Variant: "native"},
	{Name: "discord", ConfigKey: "discord"},
	{Name: "slack", ConfigKey: "slack"},
	{Name: "matrix", ConfigKey: "matrix"},
	{Name: "line", ConfigKey: "line"},
	{Name: "feishu", ConfigKey: "feishu"},
	{Name: "dingtalk", ConfigKey: "dingtalk"},
	{Name: "qq", ConfigKey: "qq"},
	{Name: "onebot", ConfigKey: "onebot"},
	{Name: "wecom", ConfigKey: "wecom"},
	{Name: "wecom_app", ConfigKey: "wecom_app"},
	{Name: "wecom_aibot", ConfigKey: "wecom_aibot"},
	{Name: "weixin", ConfigKey: "weixin"},
	{Name: "jame", ConfigKey: "jame"},
	{Name: "jame_client", ConfigKey: "jame_client"},
	{Name: "irc", ConfigKey: "irc"},
	{Name: "maixcam", ConfigKey: "maixcam"},
}

func channelCatalogItems() []extensions.CatalogItem {
	items := make([]extensions.CatalogItem, 0, len(channelCatalog))
	for _, entry := range channelCatalog {
		items = append(items, extensions.CatalogItem{
			ID:        entry.Name,
			Name:      entry.Name,
			Category:  "channel",
			ConfigKey: entry.ConfigKey,
		})
	}
	return items
}

// registerChannelRoutes binds read-only channel catalog endpoints to the ServeMux.
func (h *Handler) registerChannelRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/channels/catalog", h.handleListChannelCatalog)
}

// handleListChannelCatalog returns the channels supported by backend.
//
//	GET /api/channels/catalog
func (h *Handler) handleListChannelCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"channels": channelCatalog,
	})
}
