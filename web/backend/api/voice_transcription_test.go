package api

import (
	"bytes"
	"mime/multipart"
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

func TestVoiceTranscriptionRejectsNonAudioRecording(t *testing.T) {
	h := NewHandler("unused.yaml")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("audio", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("not audio")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/voice/transcribe", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusUnsupportedMediaType, rec.Body.String())
	}
}
