package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetHostInfo(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/system/host-info", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got hostInfoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.Hostname == "" {
		t.Fatal("hostname should not be empty")
	}
	if got.Username == "" {
		t.Fatal("username should not be empty")
	}
	if got.HomePath == "" {
		t.Fatal("home_path should not be empty")
	}
	if got.DocumentsPath == "" {
		t.Fatal("documents_path should not be empty")
	}
}
