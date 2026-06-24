package dashboard

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/web/backend/launcherconfig"
)

var (
	dashboardHTTPClient           = &http.Client{Timeout: 700 * time.Millisecond}
	dashboardOpenURL              = openURL
	dashboardCopyText             = copyText
	dashboardStartWebUI           = startWebUI
	dashboardReachable            = reachable
	dashboardWaitFor              = waitReachable
	dashboardOutput     io.Writer = os.Stdout
	dashboardWait                 = 8 * time.Second
)

func NewDashboardCommand() *cobra.Command {
	var noOpen bool

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Open the JameClaw Web Console dashboard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDashboard(cmd.Context(), dashboardOptions{NoOpen: noOpen})
		},
	}
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Print/copy the dashboard URL without opening a browser")
	return cmd
}

type dashboardOptions struct {
	NoOpen bool
}

func runDashboard(ctx context.Context, opts dashboardOptions) error {
	target, err := resolveTarget()
	if err != nil {
		return err
	}

	if !dashboardReachable(ctx, target.BaseURL) {
		fmt.Fprintf(dashboardOutput, "Web Console is not running; starting jameclaw-web on %s...\n", target.BaseURL)
		if err := dashboardStartWebUI(); err != nil {
			fmt.Fprintf(dashboardOutput, "Unable to start Web Console automatically: %v\n", err)
		} else if !dashboardWaitFor(ctx, target.BaseURL, dashboardWait) {
			fmt.Fprintf(dashboardOutput, "Web Console started but is not responding yet. Use the URL below once it is ready.\n")
		}
	}

	fmt.Fprintf(dashboardOutput, "Dashboard URL: %s\n", target.BaseURL)
	if target.AuthenticatedURL != target.BaseURL {
		fmt.Fprintln(dashboardOutput, "Access token included in copied/browser URL.")
	}
	if err := dashboardCopyText(target.AuthenticatedURL); err == nil {
		fmt.Fprintln(dashboardOutput, "Copied dashboard URL to clipboard.")
	} else {
		fmt.Fprintf(dashboardOutput, "Clipboard unavailable: %v\n", err)
	}
	fmt.Fprintf(dashboardOutput, "SSH tunnel hint: ssh -L %d:localhost:%d <host>\n", target.Port, target.Port)

	if opts.NoOpen {
		fmt.Fprintln(dashboardOutput, "Browser launch disabled by --no-open.")
		return nil
	}
	if err := dashboardOpenURL(target.AuthenticatedURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	fmt.Fprintln(dashboardOutput, "Opened dashboard in your browser.")
	return nil
}

type dashboardTarget struct {
	Port             int
	BaseURL          string
	AuthenticatedURL string
}

func resolveTarget() (dashboardTarget, error) {
	configPath := internal.GetConfigPath()
	launcherPath := launcherconfig.PathForAppConfig(configPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		return dashboardTarget{}, fmt.Errorf("failed to load launcher config: %w", err)
	}
	port := launcherCfg.Port
	if port <= 0 {
		port = launcherconfig.DefaultPort
	}

	baseURL := "http://localhost:" + strconv.Itoa(port)
	return dashboardTarget{
		Port:             port,
		BaseURL:          baseURL,
		AuthenticatedURL: authenticatedURL(baseURL),
	}, nil
}

func authenticatedURL(baseURL string) string {
	token := strings.TrimSpace(readLauncherAccessToken())
	if token == "" {
		return baseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	query := parsed.Query()
	query.Set("access_token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func readLauncherAccessToken() string {
	data, err := os.ReadFile(filepath.Join(internal.GetJameclawHome(), "launcher_access_token"))
	if err != nil {
		return ""
	}
	return string(data)
}

func reachable(ctx context.Context, rawURL string) bool {
	reqCtx, cancel := context.WithTimeout(ctx, 700*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := dashboardHTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < http.StatusInternalServerError
}

func waitReachable(ctx context.Context, rawURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if reachable(ctx, rawURL) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return reachable(ctx, rawURL)
}

func startWebUI() error {
	binary, err := resolveCompanionBinary("JAMECLAW_WEB_BINARY", "jameclaw-web")
	if err != nil {
		return err
	}
	return exec.Command(binary, "-no-browser").Start()
}

func resolveCompanionBinary(envVar, binaryName string) (string, error) {
	if custom := os.Getenv(envVar); custom != "" {
		if info, err := os.Stat(custom); err == nil && !info.IsDir() {
			return custom, nil
		}
	}

	name := binaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(exeDir, name), filepath.Join(exeDir, "build", name))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "build", name))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("%s binary not found", binaryName)
}

func openURL(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}

func copyText(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command(path)
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
