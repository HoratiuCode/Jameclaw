package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVoiceTranscriptionRequiresRecording(t *testing.T) {
	h := NewHandler("unused.yaml")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/voice/transcribe", strings.NewReader("not multipart audio"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "recording") {
		t.Fatalf("body = %q, want actionable recording error", rec.Body.String())
	}
}
