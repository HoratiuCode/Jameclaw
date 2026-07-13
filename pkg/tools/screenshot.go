package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/media"
)

const (
	jameclawScreenName      = "JameClaw Screen"
	envJameclawScreenBinary = "JAMECLAW_SCREEN_BINARY"
	envLegacyScreenBinary   = "JAMECLAW_PEEKABOO_BINARY"
)

type screenshotRequest struct {
	OutputPath  string
	Mode        string
	App         string
	WindowTitle string
	WindowIndex *int
	ScreenIndex *int
	Region      string
	Retina      bool
}

type recordingRequest struct {
	OutputPath      string
	OutputDir       string
	DurationSeconds int
	Mode            string
	App             string
	WindowTitle     string
	WindowIndex     *int
	ScreenIndex     *int
	Region          string
}

type screenshotRunner func(ctx context.Context, req screenshotRequest) error
type recordingRunner func(ctx context.Context, req recordingRequest) error

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
	return "Take a screenshot and send the PNG image to the active chat. On macOS, supports JameClaw Screen targeting for screen/window/frontmost/area captures when the JameClaw Screen binary is available. Use only when the user asks to see or share the screen."
}

func (t *ScreenshotTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional display filename. Defaults to screenshot-YYYYMMDD-HHMMSS.png.",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "Capture mode. Defaults to screen. JameClaw Screen supports screen, window, frontmost, area, and multi.",
				"enum":        []string{"screen", "window", "frontmost", "area", "multi"},
			},
			"app": map[string]any{
				"type":        "string",
				"description": "Optional macOS application target such as Safari, Google Chrome, frontmost, or menubar. Requires JameClaw Screen for app/window targeting.",
			},
			"window_title": map[string]any{
				"type":        "string",
				"description": "Optional target window title. Requires JameClaw Screen.",
			},
			"window_index": map[string]any{
				"type":        "number",
				"description": "Optional target window index for the app. Requires JameClaw Screen.",
			},
			"screen_index": map[string]any{
				"type":        "number",
				"description": "Optional 0-based display index. Requires JameClaw Screen for precise multi-display targeting.",
			},
			"region": map[string]any{
				"type":        "string",
				"description": "Optional area rectangle as x,y,width,height in global display points. Uses JameClaw Screen when available or screencapture -R on macOS.",
			},
			"retina": map[string]any{
				"type":        "boolean",
				"description": "On macOS with JameClaw Screen, capture native Retina pixels instead of logical 1x pixels.",
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

	req := screenshotRequest{
		OutputPath:  localPath,
		Mode:        cleanCaptureMode(getStringArg(args, "mode"), "screen"),
		App:         cleanCaptureValue(getStringArg(args, "app")),
		WindowTitle: cleanCaptureValue(getStringArg(args, "window_title")),
		WindowIndex: getOptionalIntArg(args, "window_index"),
		ScreenIndex: getOptionalIntArg(args, "screen_index"),
		Region:      cleanCaptureValue(getStringArg(args, "region")),
		Retina:      getOptionalBoolArg(args, "retina", false),
	}
	if req.Region != "" && getStringArg(args, "mode") == "" {
		req.Mode = "area"
	}
	if (req.App != "" || req.WindowTitle != "" || req.WindowIndex != nil) && getStringArg(args, "mode") == "" {
		req.Mode = "window"
	}

	if err := t.runner(ctx, req); err != nil {
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

func captureScreenshot(ctx context.Context, req screenshotRequest) error {
	switch runtime.GOOS {
	case "darwin":
		if canUseJameclawScreen() {
			return captureScreenshotWithJameclawScreen(ctx, req)
		}
		return captureDarwinScreenshotFallback(ctx, req)
	case "linux":
		return captureLinuxScreenshot(ctx, req.OutputPath)
	case "windows":
		return captureWindowsScreenshot(ctx, req.OutputPath)
	default:
		return fmt.Errorf("screenshots are not supported on %s", runtime.GOOS)
	}
}

func captureScreenshotWithJameclawScreen(ctx context.Context, req screenshotRequest) error {
	binary, err := findJameclawScreenBinary()
	if err != nil {
		return err
	}
	args := []string{"image", "--path", req.OutputPath, "--mode", req.Mode}
	if req.App != "" {
		args = append(args, "--app", req.App)
	}
	if req.WindowTitle != "" {
		args = append(args, "--window-title", req.WindowTitle)
	}
	if req.WindowIndex != nil {
		args = append(args, "--window-index", strconv.Itoa(*req.WindowIndex))
	}
	if req.ScreenIndex != nil {
		args = append(args, "--screen-index", strconv.Itoa(*req.ScreenIndex))
	}
	if req.Region != "" {
		args = append(args, "--region", req.Region)
	}
	if req.Retina {
		args = append(args, "--retina")
	}
	if req.App != "" || req.WindowTitle != "" || req.WindowIndex != nil {
		args = append(args, "--capture-focus", "background")
	}
	return runCaptureCommand(ctx, binary, args...)
}

func captureDarwinScreenshotFallback(ctx context.Context, req screenshotRequest) error {
	if req.App != "" || req.WindowTitle != "" || req.WindowIndex != nil || req.ScreenIndex != nil || req.Retina || req.Mode == "window" || req.Mode == "frontmost" || req.Mode == "multi" {
		return fmt.Errorf("targeted macOS screenshots require %s; set %s to the JameClaw Screen binary", jameclawScreenName, envJameclawScreenBinary)
	}
	args := []string{"-x"}
	if req.Region != "" {
		args = append(args, "-R", req.Region)
	}
	args = append(args, req.OutputPath)
	return runCaptureCommand(ctx, "screencapture", args...)
}

type ScreenRecordingTool struct {
	maxFileSize int
	mediaStore  media.MediaStore
	runner      recordingRunner
	now         func() time.Time
}

func NewScreenRecordingTool(maxFileSize int, store media.MediaStore) *ScreenRecordingTool {
	if maxFileSize <= 0 {
		maxFileSize = config.DefaultMaxMediaSize
	}
	return &ScreenRecordingTool{
		maxFileSize: maxFileSize,
		mediaStore:  store,
		runner:      captureScreenRecording,
		now:         time.Now,
	}
}

func (t *ScreenRecordingTool) Name() string { return "screen_recording" }

func (t *ScreenRecordingTool) Description() string {
	return "Record the screen and send the video to the active chat. On macOS, uses JameClaw Screen live capture when available for targetable app/window/screen/area recording, with screencapture fallback for full-screen recordings."
}

func (t *ScreenRecordingTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional display filename. Defaults to recording-YYYYMMDD-HHMMSS.mp4 with JameClaw Screen or .mov with fallback.",
			},
			"duration_seconds": map[string]any{
				"type":        "number",
				"description": "Recording duration in seconds. Defaults to 5. Maximum 180.",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "Capture mode. Defaults to screen. JameClaw Screen supports screen, window, frontmost, and area.",
				"enum":        []string{"screen", "window", "frontmost", "area"},
			},
			"app": map[string]any{"type": "string", "description": "Optional macOS application target. Requires JameClaw Screen."},
			"window_title": map[string]any{
				"type":        "string",
				"description": "Optional target window title. Requires JameClaw Screen.",
			},
			"window_index": map[string]any{"type": "number", "description": "Optional target window index. Requires JameClaw Screen."},
			"screen_index": map[string]any{"type": "number", "description": "Optional 0-based display index. Requires JameClaw Screen."},
			"region": map[string]any{
				"type":        "string",
				"description": "Optional area rectangle as x,y,width,height in global display points. Requires JameClaw Screen.",
			},
		},
	}
}

func (t *ScreenRecordingTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *ScreenRecordingTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat available")
	}
	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}
	if t.runner == nil {
		t.runner = captureScreenRecording
	}
	if t.now == nil {
		t.now = time.Now
	}
	if err := os.MkdirAll(media.TempDir(), 0o700); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create media directory: %v", err)).WithError(err)
	}

	duration := getOptionalIntArgValue(args, "duration_seconds", 5)
	if duration < 1 {
		duration = 1
	}
	if duration > 180 {
		duration = 180
	}

	useJameclawScreen := runtime.GOOS == "darwin" && canUseJameclawScreen()
	defaultExt := ".mov"
	contentType := "video/quicktime"
	if useJameclawScreen {
		defaultExt = ".mp4"
		contentType = "video/mp4"
	}
	filename := strings.TrimSpace(getStringArg(args, "filename"))
	if filename == "" {
		filename = "recording-" + t.now().Format("20060102-150405") + defaultExt
	}
	filename = ensureVideoFilename(sanitizeScreenshotFilename(filename), defaultExt)
	contentType = videoContentType(filename)
	localPath := filepath.Join(media.TempDir(), uuid.New().String()+"-"+filename)
	outputDir := filepath.Join(media.TempDir(), uuid.New().String()+"-recording")

	req := recordingRequest{
		OutputPath:      localPath,
		OutputDir:       outputDir,
		DurationSeconds: duration,
		Mode:            cleanCaptureMode(getStringArg(args, "mode"), "screen"),
		App:             cleanCaptureValue(getStringArg(args, "app")),
		WindowTitle:     cleanCaptureValue(getStringArg(args, "window_title")),
		WindowIndex:     getOptionalIntArg(args, "window_index"),
		ScreenIndex:     getOptionalIntArg(args, "screen_index"),
		Region:          cleanCaptureValue(getStringArg(args, "region")),
	}
	if req.Region != "" && getStringArg(args, "mode") == "" {
		req.Mode = "area"
	}
	if (req.App != "" || req.WindowTitle != "" || req.WindowIndex != nil) && getStringArg(args, "mode") == "" {
		req.Mode = "window"
	}

	if err := t.runner(ctx, req); err != nil {
		_ = os.Remove(localPath)
		_ = os.RemoveAll(outputDir)
		return ErrorResult(fmt.Sprintf("failed to record screen: %v", err)).WithError(err)
	}
	defer os.RemoveAll(outputDir)

	info, err := os.Stat(localPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("recording file was not created: %v", err)).WithError(err)
	}
	if info.IsDir() {
		return ErrorResult("recording output is a directory")
	}
	if info.Size() == 0 {
		_ = os.Remove(localPath)
		return ErrorResult("screen recording produced an empty file")
	}
	if info.Size() > int64(t.maxFileSize) {
		_ = os.Remove(localPath)
		return ErrorResult(fmt.Sprintf("screen recording too large: %d bytes (max %d bytes)", info.Size(), t.maxFileSize))
	}

	scope := fmt.Sprintf("tool:screen_recording:%s:%s", channel, chatID)
	ref, err := t.mediaStore.Store(localPath, media.MediaMeta{
		Filename:      filename,
		ContentType:   contentType,
		Source:        "tool:screen_recording",
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		_ = os.Remove(localPath)
		return ErrorResult(fmt.Sprintf("failed to register screen recording: %v", err)).WithError(err)
	}
	return MediaResult(fmt.Sprintf("Screen recording %q captured and sent to user", filename), []string{ref})
}

func captureScreenRecording(ctx context.Context, req recordingRequest) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("screen recording is currently supported on macOS only")
	}
	if canUseJameclawScreen() {
		return captureScreenRecordingWithJameclawScreen(ctx, req)
	}
	if req.App != "" || req.WindowTitle != "" || req.WindowIndex != nil || req.ScreenIndex != nil || req.Region != "" || req.Mode != "screen" {
		return fmt.Errorf("targeted macOS screen recording requires %s; set %s to the JameClaw Screen binary", jameclawScreenName, envJameclawScreenBinary)
	}
	return runCaptureCommand(ctx, "screencapture", "-v", "-x", "-T", strconv.Itoa(req.DurationSeconds), req.OutputPath)
}

func captureScreenRecordingWithJameclawScreen(ctx context.Context, req recordingRequest) error {
	binary, err := findJameclawScreenBinary()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(req.OutputDir, 0o700); err != nil {
		return err
	}
	args := []string{
		"capture", "live",
		"--duration", strconv.Itoa(req.DurationSeconds),
		"--mode", req.Mode,
		"--path", req.OutputDir,
		"--video-out", req.OutputPath,
		"--capture-focus", "background",
	}
	if req.App != "" {
		args = append(args, "--app", req.App)
	}
	if req.WindowTitle != "" {
		args = append(args, "--window-title", req.WindowTitle)
	}
	if req.WindowIndex != nil {
		args = append(args, "--window-index", strconv.Itoa(*req.WindowIndex))
	}
	if req.ScreenIndex != nil {
		args = append(args, "--screen-index", strconv.Itoa(*req.ScreenIndex))
	}
	if req.Region != "" {
		args = append(args, "--region", req.Region)
	}
	return runCaptureCommand(ctx, binary, args...)
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
		return runCaptureCommand(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	}
	if _, err := exec.LookPath("pwsh"); err == nil {
		return runCaptureCommand(ctx, "pwsh", "-NoProfile", "-NonInteractive", "-Command", script)
	}
	return fmt.Errorf("PowerShell is required to capture screenshots on Windows")
}

func runScreenshotCommand(ctx context.Context, command string, args ...string) error {
	return runCaptureCommand(ctx, command, args...)
}

func runCaptureCommand(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		msg = improveCaptureErrorMessage(msg)
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

func ensureVideoFilename(filename, defaultExt string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".mp4" || ext == ".mov" {
		return filename
	}
	return strings.TrimSuffix(filename, filepath.Ext(filename)) + defaultExt
}

func videoContentType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}

func powershellStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func cleanCaptureValue(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\x00", ""), "\n", " "))
}

func cleanCaptureMode(value, fallback string) string {
	value = strings.ToLower(cleanCaptureValue(value))
	switch value {
	case "screen", "window", "frontmost", "area", "multi":
		return value
	case "":
		return fallback
	default:
		return fallback
	}
}

func getOptionalIntArg(args map[string]any, key string) *int {
	value, ok := getNumberArg(args, key)
	if !ok {
		return nil
	}
	intValue := int(value)
	return &intValue
}

func getOptionalIntArgValue(args map[string]any, key string, fallback int) int {
	value := getOptionalIntArg(args, key)
	if value == nil {
		return fallback
	}
	return *value
}

func canUseJameclawScreen() bool {
	_, err := findJameclawScreenBinary()
	return err == nil
}

func findJameclawScreenBinary() (string, error) {
	if p := strings.TrimSpace(os.Getenv(envJameclawScreenBinary)); p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("%s does not point to an executable file", envJameclawScreenBinary)
	}
	if p := strings.TrimSpace(os.Getenv(envLegacyScreenBinary)); p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("%s does not point to an executable file", envLegacyScreenBinary)
	}
	if p, err := exec.LookPath("peekaboo"); err == nil {
		return p, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, "Downloads", "Peekaboo-main", "Apps", "peekaboo")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s binary not found", jameclawScreenName)
}

func improveCaptureErrorMessage(msg string) string {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "screen recording") || strings.Contains(lower, "not authorized") || strings.Contains(lower, "permission") || strings.Contains(lower, "denied") {
		return msg + "; grant Screen Recording permission in System Settings > Privacy & Security > Screen & System Audio Recording"
	}
	return msg
}
