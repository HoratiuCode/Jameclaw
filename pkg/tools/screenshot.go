package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/media"
)

type screenshotRunner func(ctx context.Context, outputPath string) error

// ScreenshotTool captures the visible desktop and returns it as outbound media.
type ScreenshotTool struct {
	maxFileSize int
	mediaStore  media.MediaStore
	runner      screenshotRunner
	now         func() time.Time
}

func NewScreenshotTool(maxFileSize int, store media.MediaStore) *ScreenshotTool {
	if maxFileSize <= 0 {
		maxFileSize = config.DefaultMaxMediaSize
	}
	return &ScreenshotTool{
		maxFileSize: maxFileSize,
		mediaStore:  store,
		runner:      captureScreenshot,
		now:         time.Now,
	}
}

func (t *ScreenshotTool) Name() string { return "screenshot" }

func (t *ScreenshotTool) Description() string {
	return "Take a screenshot of the current visible desktop and send the PNG image to the active chat. Use only when the user asks to see or share the screen."
}

func (t *ScreenshotTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional display filename. Defaults to screenshot-YYYYMMDD-HHMMSS.png.",
			},
		},
	}
}

func (t *ScreenshotTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *ScreenshotTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat available")
	}
	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}
	if t.runner == nil {
		t.runner = captureScreenshot
	}
	if t.now == nil {
		t.now = time.Now
	}

	if err := os.MkdirAll(media.TempDir(), 0o700); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create media directory: %v", err)).WithError(err)
	}

	filename := strings.TrimSpace(getStringArg(args, "filename"))
	if filename == "" {
		filename = "screenshot-" + t.now().Format("20060102-150405") + ".png"
	}
	filename = ensurePNGFilename(sanitizeScreenshotFilename(filename))
	localPath := filepath.Join(media.TempDir(), uuid.New().String()+"-"+filename)

	if err := t.runner(ctx, localPath); err != nil {
		_ = os.Remove(localPath)
		return ErrorResult(fmt.Sprintf("failed to capture screenshot: %v", err)).WithError(err)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("screenshot file was not created: %v", err)).WithError(err)
	}
	if info.IsDir() {
		return ErrorResult("screenshot output is a directory")
	}
	if info.Size() == 0 {
		_ = os.Remove(localPath)
		return ErrorResult("screenshot capture produced an empty file")
	}
	if info.Size() > int64(t.maxFileSize) {
		_ = os.Remove(localPath)
		return ErrorResult(fmt.Sprintf(
			"screenshot too large: %d bytes (max %d bytes)",
			info.Size(), t.maxFileSize,
		))
	}

	scope := fmt.Sprintf("tool:screenshot:%s:%s", channel, chatID)
	ref, err := t.mediaStore.Store(localPath, media.MediaMeta{
		Filename:      filename,
		ContentType:   "image/png",
		Source:        "tool:screenshot",
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		_ = os.Remove(localPath)
		return ErrorResult(fmt.Sprintf("failed to register screenshot: %v", err)).WithError(err)
	}

	return MediaResult(fmt.Sprintf("Screenshot %q captured and sent to user", filename), []string{ref})
}

func captureScreenshot(ctx context.Context, outputPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return runScreenshotCommand(ctx, "screencapture", "-x", outputPath)
	case "linux":
		return captureLinuxScreenshot(ctx, outputPath)
	case "windows":
		return captureWindowsScreenshot(ctx, outputPath)
	default:
		return fmt.Errorf("screenshots are not supported on %s", runtime.GOOS)
	}
}

func captureLinuxScreenshot(ctx context.Context, outputPath string) error {
	candidates := [][]string{
		{"gnome-screenshot", "-f", outputPath},
		{"grim", outputPath},
		{"spectacle", "-b", "-n", "-o", outputPath},
		{"import", "-window", "root", outputPath},
	}
	var missing []string
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			missing = append(missing, candidate[0])
			continue
		}
		return runScreenshotCommand(ctx, candidate[0], candidate[1:]...)
	}
	return fmt.Errorf("no supported screenshot command found; tried %s", strings.Join(missing, ", "))
}

func captureWindowsScreenshot(ctx context.Context, outputPath string) error {
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bitmap = New-Object System.Drawing.Bitmap $bounds.Width, $bounds.Height; $graphics = [System.Drawing.Graphics]::FromImage($bitmap); $graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size); $bitmap.Save(%s, [System.Drawing.Imaging.ImageFormat]::Png); $graphics.Dispose(); $bitmap.Dispose()`, powershellStringLiteral(outputPath))
	if _, err := exec.LookPath("powershell"); err == nil {
		return runScreenshotCommand(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	}
	if _, err := exec.LookPath("pwsh"); err == nil {
		return runScreenshotCommand(ctx, "pwsh", "-NoProfile", "-NonInteractive", "-Command", script)
	}
	return fmt.Errorf("PowerShell is required to capture screenshots on Windows")
}

func runScreenshotCommand(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s: %s", command, msg)
	}
	return nil
}

func sanitizeScreenshotFilename(filename string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "." || filename == string(os.PathSeparator) || filename == "" {
		return "screenshot.png"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "\x00", "_")
	return replacer.Replace(filename)
}

func ensurePNGFilename(filename string) string {
	if strings.EqualFold(filepath.Ext(filename), ".png") {
		return filename
	}
	return strings.TrimSuffix(filename, filepath.Ext(filename)) + ".png"
}

func powershellStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
