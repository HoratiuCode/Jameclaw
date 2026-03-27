package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
)

var launcherAccessToken string

func initLauncherAccessToken() error {
	token, err := generateLauncherAccessToken()
	if err != nil {
		return err
	}
	launcherAccessToken = token
	return nil
}

func generateLauncherAccessToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate launcher access token: %w", err)
	}
	return hex.EncodeToString(buf), nil
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
