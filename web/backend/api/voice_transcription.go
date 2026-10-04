package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/HoratiuCode/jameclaw/pkg/config"
	"github.com/HoratiuCode/jameclaw/pkg/utils"
	"github.com/HoratiuCode/jameclaw/pkg/voice"
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

	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "The request does not include an audio recording.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Do not force every multipart recording to .m4a. The web console sends
	// WebM/Opus in Chromium and the extension is how the provider serializer
	// determines its audio format.
	ext := strings.ToLower(filepath.Ext(utils.SanitizeFilename(header.Filename)))
	if !utils.IsAudioFile(header.Filename, header.Header.Get("Content-Type")) {
		http.Error(w, "The recording must be a supported audio format.", http.StatusUnsupportedMediaType)
		return
	}
	if ext == "" {
		ext = ".m4a"
	}
	temporary, err := os.CreateTemp("", "jameclaw-voice-*"+ext)
	if err != nil {
		http.Error(w, "Could not prepare the recording for transcription.", http.StatusInternalServerError)
		return
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	written, err := io.Copy(temporary, io.LimitReader(file, maxVoiceRecordingBytes+1))
	if err != nil {
		temporary.Close()
		http.Error(w, "Could not read the recording.", http.StatusBadRequest)
		return
	}
	if written > maxVoiceRecordingBytes {
		temporary.Close()
		http.Error(w, "The recording is larger than 25 MB.", http.StatusRequestEntityTooLarge)
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
