package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/providers/common"
)

// GeneratedImage is one image returned by an image generation provider.
// Data contains decoded image bytes when the provider responds with b64_json;
// otherwise URL contains the temporary download URL returned by the provider.
type GeneratedImage struct {
	Data        []byte
	URL         string
	ContentType string
}

// GenerateImage calls the OpenAI-compatible images generation endpoint for a
// configured model. Providers without this endpoint return their API error, so
// callers can continue through configured image-model fallbacks.
func GenerateImage(ctx context.Context, cfg *config.ModelConfig, prompt, size, quality string) (*GeneratedImage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("image model is not configured")
	}
	protocol, model := ExtractProtocol(cfg.Model)
	if !supportsImageGenerationProtocol(protocol) {
		return nil, fmt.Errorf("provider %q does not expose an OpenAI-compatible image generation API", protocol)
	}
	apiBase := strings.TrimRight(cfg.APIBase, "/")
	if apiBase == "" {
		apiBase = getDefaultAPIBase(protocol)
	}
	if apiBase == "" {
		return nil, fmt.Errorf("API base is not configured for provider %q", protocol)
	}

	body := map[string]any{"model": model, "prompt": prompt, "n": 1, "response_format": "b64_json"}
	if size != "" {
		body["size"] = size
	}
	if quality != "" {
		body["quality"] = quality
	}
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode image request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/images/generations", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create image request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := cfg.APIKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := common.NewHTTPClient(cfg.Proxy)
	if cfg.RequestTimeout > 0 {
		client.Timeout = time.Duration(cfg.RequestTimeout) * time.Second
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send image request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return nil, fmt.Errorf("image generation failed (%s): %s", resp.Status, strings.TrimSpace(string(message)))
	}
	var result struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode image response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("image generation response contained no images")
	}
	image := result.Data[0]
	if image.B64JSON != "" {
		data, err := base64.StdEncoding.DecodeString(image.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decode generated image: %w", err)
		}
		return &GeneratedImage{Data: data, ContentType: "image/png"}, nil
	}
	if image.URL != "" {
		return &GeneratedImage{URL: image.URL}, nil
	}
	return nil, fmt.Errorf("image generation response contained neither image data nor URL")
}

func supportsImageGenerationProtocol(protocol string) bool {
	switch protocol {
	case "openai", "litellm", "openrouter", "groq", "xai", "zhipu", "gemini", "nvidia", "ollama", "moonshot", "shengsuanyun", "deepseek", "cerebras", "vivgrid", "volcengine", "vllm", "qwen", "qwen-intl", "qwen-international", "dashscope-intl", "qwen-us", "dashscope-us", "mistral", "avian", "longcat", "modelscope", "novita", "nous", "coding-plan", "alibaba-coding", "qwen-coding":
		return true
	default:
		return false
	}
}
