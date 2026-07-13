package voice

import (
	"context"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/providers"
)

type Transcriber interface {
	Name() string
	Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error)
}

type TranscriptionResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

func supportsAudioTranscription(model string) bool {
	protocol, _ := providers.ExtractProtocol(model)

	switch protocol {
	case "openai", "azure", "azure-openai",
		"litellm", "openrouter", "groq", "zhipu", "gemini", "nvidia",
		"ollama", "moonshot", "shengsuanyun", "deepseek", "cerebras",
		"vivgrid", "volcengine", "vllm", "qwen", "qwen-intl", "qwen-international", "dashscope-intl",
		"qwen-us", "dashscope-us", "mistral", "avian", "minimax", "longcat", "modelscope", "novita",
		"coding-plan", "alibaba-coding", "qwen-coding", "codex-cli", "codexcli":
		// These protocols can receive the audio media payload shape used by
		// NewAudioModelTranscriber. HTTP-compatible providers serialize audio as
		// input_audio; the Codex CLI provider stages audio into local files and
		// references those files from the prompt.

		// TODO: Further restrict this by modelID, since not every model under these
		// protocols supports audio transcription.
		return true
	default:
		return false
	}
}

// DetectTranscriber inspects cfg and returns the appropriate Transcriber, or
// nil if no supported transcription provider is configured.
func DetectTranscriber(cfg *config.Config) Transcriber {
	if modelName := strings.TrimSpace(cfg.Voice.ModelName); modelName != "" {
		modelCfg, err := cfg.GetModelConfig(modelName)
		if err != nil {
			return nil
		}
		if supportsAudioTranscription(modelCfg.Model) {
			return NewAudioModelTranscriber(modelCfg)
		}
	}

	if modelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName()); modelName != "" {
		modelCfg, err := cfg.GetModelConfig(modelName)
		if err == nil && supportsAudioTranscription(modelCfg.Model) {
			if transcriber := NewAudioModelTranscriber(modelCfg); transcriber != nil {
				return transcriber
			}
		}
	}

	// ElevenLabs voice config (supports Scribe STT).
	if key := strings.TrimSpace(cfg.Voice.ElevenLabsAPIKey); key != "" {
		return NewElevenLabsTranscriber(key)
	}
	// Fall back to any model-list entry that uses the groq/ protocol.
	for _, mc := range cfg.ModelList {
		if strings.HasPrefix(mc.Model, "groq/") && mc.APIKey() != "" {
			return NewGroqTranscriber(mc.APIKey())
		}
	}
	return nil
}
