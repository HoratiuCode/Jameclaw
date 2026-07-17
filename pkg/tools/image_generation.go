package tools

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/media"
	"github.com/sipeed/jameclaw/pkg/providers"
	"github.com/sipeed/jameclaw/pkg/providers/common"
)

// ImageGenerationTool generates an image with the configured image model and
// returns it through JameClaw's normal outbound-media pipeline.
type ImageGenerationTool struct {
	cfg         *config.Config
	maxFileSize int
	mediaStore  media.MediaStore
}

func NewImageGenerationTool(cfg *config.Config, maxFileSize int, store media.MediaStore) *ImageGenerationTool {
	if maxFileSize <= 0 {
		maxFileSize = config.DefaultMaxMediaSize
	}
	return &ImageGenerationTool{cfg: cfg, maxFileSize: maxFileSize, mediaStore: store}
}

func (t *ImageGenerationTool) Name() string { return "image_generation" }

func (t *ImageGenerationTool) Description() string {
	return "Generate an image from a prompt using the configured image model and send it to the active chat. Use this when the user explicitly asks to create, generate, draw, or make an image."
}

func (t *ImageGenerationTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt":   map[string]any{"type": "string", "description": "Detailed description of the image to generate."},
			"size":     map[string]any{"type": "string", "description": "Optional provider-supported size, such as 1024x1024."},
			"quality":  map[string]any{"type": "string", "description": "Optional provider-supported quality setting, such as low, medium, or high."},
			"filename": map[string]any{"type": "string", "description": "Optional display filename. Defaults to generated-image.png."},
		},
		"required": []string{"prompt"},
	}
}

func (t *ImageGenerationTool) SetMediaStore(store media.MediaStore) { t.mediaStore = store }

func (t *ImageGenerationTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	prompt := strings.TrimSpace(getStringArg(args, "prompt"))
	if prompt == "" {
		return ErrorResult("prompt is required")
	}
	channel, chatID := ToolChannel(ctx), ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat available")
	}
	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}
	models := t.imageModels()
	if len(models) == 0 {
		return ErrorResult("no image model is configured; set agents.defaults.image_model to a model_name in model_list")
	}

	filename := sanitizeImageFilename(getStringArg(args, "filename"))
	if filename == "" {
		filename = "generated-image.png"
	}
	var failures []string
	for _, model := range models {
		generated, err := providers.GenerateImage(ctx, model, prompt, getStringArg(args, "size"), getStringArg(args, "quality"))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", model.ModelName, err))
			continue
		}
		path, contentType, err := t.saveGeneratedImage(ctx, generated, filename, model.Proxy)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", model.ModelName, err))
			continue
		}
		if contentType == "" {
			contentType = "image/png"
		}
		scope := fmt.Sprintf("tool:image_generation:%s:%s", channel, chatID)
		ref, err := t.mediaStore.Store(path, media.MediaMeta{
			Filename: filename, ContentType: contentType, Source: "tool:image_generation", CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
		}, scope)
		if err != nil {
			_ = os.Remove(path)
			return ErrorResult(fmt.Sprintf("failed to register generated image: %v", err)).WithError(err)
		}
		return MediaResult(fmt.Sprintf("Image generated with %q and sent to the user", model.ModelName), []string{ref})
	}
	return ErrorResult("image generation failed for all configured image models: " + strings.Join(failures, "; "))
}

func (t *ImageGenerationTool) imageModels() []*config.ModelConfig {
	if t.cfg == nil {
		return nil
	}
	names := append([]string{t.cfg.Agents.Defaults.ImageModel}, t.cfg.Agents.Defaults.ImageModelFallbacks...)
	seen := make(map[string]bool, len(names))
	models := make([]*config.ModelConfig, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		for _, model := range t.cfg.ModelList {
			if model != nil && model.ModelName == name {
				models = append(models, model)
				break
			}
		}
	}
	return models
}

func (t *ImageGenerationTool) saveGeneratedImage(ctx context.Context, image *providers.GeneratedImage, filename, proxy string) (string, string, error) {
	if image == nil {
		return "", "", fmt.Errorf("provider returned no image")
	}
	data, contentType := image.Data, image.ContentType
	if len(data) == 0 && image.URL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, image.URL, nil)
		if err != nil {
			return "", "", fmt.Errorf("create image download request: %w", err)
		}
		resp, err := common.NewHTTPClient(proxy).Do(req)
		if err != nil {
			return "", "", fmt.Errorf("download generated image: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", "", fmt.Errorf("download generated image: %s", resp.Status)
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, int64(t.maxFileSize)+1))
		if err != nil {
			return "", "", fmt.Errorf("read generated image: %w", err)
		}
		contentType = resp.Header.Get("Content-Type")
	}
	if len(data) == 0 {
		return "", "", fmt.Errorf("generated image is empty")
	}
	if len(data) > t.maxFileSize {
		return "", "", fmt.Errorf("generated image too large: %d bytes (max %d bytes)", len(data), t.maxFileSize)
	}
	if err := os.MkdirAll(media.TempDir(), 0o700); err != nil {
		return "", "", fmt.Errorf("create media directory: %w", err)
	}
	path := filepath.Join(media.TempDir(), uuid.New().String()+"-"+filename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", "", fmt.Errorf("write generated image: %w", err)
	}
	return path, contentType, nil
}

func sanitizeImageFilename(filename string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	filename = strings.ReplaceAll(filename, "\x00", "")
	if filename == "." || filename == "/" {
		return ""
	}
	if ext := strings.ToLower(filepath.Ext(filename)); ext == "" {
		filename += ".png"
	} else if mime.TypeByExtension(ext) == "" {
		filename += ".png"
	}
	return filename
}
