package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/jameclaw/web/backend/utils"
)

var launcherAccessToken string

const launcherAccessTokenFile = "launcher_access_token"

func initLauncherAccessToken() error {
	token, err := generateLauncherAccessToken()
	if err != nil {
		return err
	}
	launcherAccessToken = token
	if err := persistLauncherAccessToken(token); err != nil {
		return err
	}
	return nil
}

func generateLauncherAccessToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate launcher access token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func persistLauncherAccessToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("launcher access token is empty")
	}

	jameHome := utils.GetJameclawHome()
	if err := os.MkdirAll(jameHome, 0o700); err != nil {
		return fmt.Errorf("create jameclaw home: %w", err)
	}
	path := filepath.Join(jameHome, launcherAccessTokenFile)
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("write launcher access token: %w", err)
	}
	return nil
}

func launcherOpenURL(baseURL string) string {
	if baseURL == "" || launcherAccessToken == "" {
		return baseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	query := parsed.Query()
	query.Set("access_token", launcherAccessToken)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
