package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sipeed/jameclaw/pkg/config"
)

// GetJameclawHome returns the jameclaw home directory.
// Priority: $JAMECLAW_HOME > ~/.jameclaw
func GetJameclawHome() string {
	if home := os.Getenv(config.EnvHome); home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".jameclaw")
}

// GetDefaultConfigPath returns the default path to the jameclaw config file.
func GetDefaultConfigPath() string {
	if configPath := os.Getenv(config.EnvConfig); configPath != "" {
		return configPath
	}
	return filepath.Join(GetJameclawHome(), "config.json")
}

// FindJameclawBinary locates the jameclaw executable.
// Search order:
//  1. JAMECLAW_BINARY environment variable (explicit override)
//  2. Same directory as the current executable
//  3. Falls back to "jameclaw" and relies on $PATH
func FindJameclawBinary() string {
	binaryName := "jameclaw"
	if runtime.GOOS == "windows" {
		binaryName = "jameclaw.exe"
	}

	if p := os.Getenv(config.EnvBinary); p != "" {
		if info, _ := os.Stat(p); info != nil && !info.IsDir() {
			return p
		}
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), binaryName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return "jameclaw"
}

// GetLocalIP returns the local IP address of the machine.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}

// OpenBrowser automatically opens the given URL in the default browser.
func OpenBrowser(url string) error {
	command, args, err := openBrowserCommand(runtime.GOOS, url, false)
	if err != nil {
		return err
	}
	return exec.Command(command, args...).Start()
}

// OpenBrowserBackground opens the given URL without activating the browser when supported.
func OpenBrowserBackground(url string) error {
	command, args, err := openBrowserCommand(runtime.GOOS, url, true)
	if err != nil {
		return err
	}
	return exec.Command(command, args...).Start()
}

func openBrowserCommand(goos, url string, background bool) (string, []string, error) {
	switch goos {
	case "linux":
		return "xdg-open", []string{url}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	case "darwin":
		if background {
			return "open", []string{"-g", url}, nil
		}
		return "open", []string{url}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform")
	}
}
