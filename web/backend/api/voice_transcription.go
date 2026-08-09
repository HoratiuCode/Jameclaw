package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/voice"
)

const maxVoiceRecordingBytes = 25 << 20

func (h *Handler) registerVoiceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/voice/transcribe", h.handleVoiceTranscription)
}

func (h *Handler) handleVoiceTranscription(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxVoiceRecordingBytes)
	if err := r.ParseMultipartForm(maxVoiceRecordingBytes); err != nil {
		http.Error(w, "The recording is invalid or larger than 25 MB.", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "The request does not include an audio recording.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	temporary, err := os.CreateTemp("", "jameclaw-voice-*.m4a")
	if err != nil {
		http.Error(w, "Could not prepare the recording for transcription.", http.StatusInternalServerError)
		return
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := io.Copy(temporary, io.LimitReader(file, maxVoiceRecordingBytes+1)); err != nil {
		temporary.Close()
		http.Error(w, "Could not read the recording.", http.StatusBadRequest)
		return
	}
	if err := temporary.Close(); err != nil {
		http.Error(w, "Could not prepare the recording for transcription.", http.StatusInternalServerError)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Could not load the voice transcription settings.", http.StatusInternalServerError)
		return
	}
	transcriber := voice.DetectTranscriber(cfg)
	if transcriber == nil {
		http.Error(w, "No voice transcription model is configured. Choose a voice-capable model in Settings, then try again.", http.StatusServiceUnavailable)
		return
	}

	result, err := transcriber.Transcribe(r.Context(), temporaryPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Voice transcription failed with %s. Check that model's connection and try again.", transcriber.Name()), http.StatusBadGateway)
		return
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		http.Error(w, "The transcription was empty. Try recording again in a quieter place.", http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
}
