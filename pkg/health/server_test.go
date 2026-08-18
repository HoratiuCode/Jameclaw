package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReportsAgentActive(t *testing.T) {
	server := NewServer("127.0.0.1", 0)
	server.SetAgentActiveFunc(func() bool { return true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	server.healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.AgentActive {
		t.Fatal("AgentActive = false, want true")
	}
}

func TestHealthReportsLiveChannelStatus(t *testing.T) {
	server := NewServer("127.0.0.1", 0)
	server.SetChannelStatusFunc(func() map[string]any {
		return map[string]any{
			"telegram": map[string]any{"enabled": true, "running": true},
		}
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	server.healthHandler(rec, req)

	var body StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	telegram, ok := body.Channels["telegram"].(map[string]any)
	if !ok || telegram["running"] != true {
		t.Fatalf("telegram status = %#v, want running", body.Channels["telegram"])
	}
}
