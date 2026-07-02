package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/web/backend/utils"
)

const (
	githubLatestReleaseURL = "https://api.github.com/repos/sipeed/jameclaw/releases/latest"
	githubReleasesPageURL  = "https://github.com/sipeed/jameclaw/releases/latest"
	updateStateFile        = "update_state.json"
	updateCheckTimeout     = 5 * time.Second
)

var versionPartPattern = regexp.MustCompile(`\d+`)

type updateStatusResponse struct {
	CurrentVersion   string `json:"current_version"`
	LatestVersion    string `json:"latest_version,omitempty"`
	LatestName       string `json:"latest_name,omitempty"`
	ReleaseURL       string `json:"release_url,omitempty"`
	PublishedAt      string `json:"published_at,omitempty"`
	UpdateAvailable  bool   `json:"update_available"`
	Dismissed        bool   `json:"dismissed"`
	CheckError       string `json:"check_error,omitempty"`
	UpdateAction     string `json:"update_action"`
	UpdateActionText string `json:"update_action_text"`
}

type githubReleaseResponse struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

type updateState struct {
	DismissedVersion string `json:"dismissed_version,omitempty"`
}

type dismissUpdateRequest struct {
	Version string `json:"version"`
}

func (h *Handler) registerUpdateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/update/status", h.handleUpdateStatus)
	mux.HandleFunc("POST /api/update/dismiss", h.handleUpdateDismiss)
	mux.HandleFunc("POST /api/update/open", h.handleUpdateOpen)
}

func (h *Handler) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	status := h.getUpdateStatus(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (h *Handler) handleUpdateDismiss(w http.ResponseWriter, r *http.Request) {
	var req dismissUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if err := saveUpdateState(updateState{DismissedVersion: version}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save update dismissal: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "dismissed"})
}

func (h *Handler) handleUpdateOpen(w http.ResponseWriter, r *http.Request) {
	status := h.getUpdateStatus(r.Context())
	targetURL := strings.TrimSpace(status.ReleaseURL)
	if targetURL == "" {
		targetURL = githubReleasesPageURL
	}
	if err := utils.OpenBrowser(targetURL); err != nil {
		http.Error(w, fmt.Sprintf("Failed to open update page: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "opened",
		"release_url": targetURL,
	})
}

func (h *Handler) getUpdateStatus(ctx context.Context) updateStatusResponse {
	current := strings.TrimSpace(config.GetVersion())
	state, _ := loadUpdateState()
	status := updateStatusResponse{
		CurrentVersion:   current,
		ReleaseURL:       githubReleasesPageURL,
		UpdateAction:     "open_release",
		UpdateActionText: "Update",
	}

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		status.CheckError = err.Error()
		return status
	}
	latest := strings.TrimSpace(release.TagName)
	if latest == "" {
		latest = strings.TrimSpace(release.Name)
	}
	if latest == "" {
		status.CheckError = "latest release did not include a version"
		return status
	}

	status.LatestVersion = latest
	status.LatestName = strings.TrimSpace(release.Name)
	status.PublishedAt = strings.TrimSpace(release.PublishedAt)
	if strings.TrimSpace(release.HTMLURL) != "" {
		status.ReleaseURL = strings.TrimSpace(release.HTMLURL)
	}
	status.UpdateAvailable = isVersionNewer(latest, current)
	status.Dismissed = status.UpdateAvailable && state.DismissedVersion == latest
	return status
}

func fetchLatestRelease(ctx context.Context) (githubReleaseResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, updateCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseURL, nil)
	if err != nil {
		return githubReleaseResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", fmt.Sprintf("jameclaw/%s", config.GetVersion()))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubReleaseResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubReleaseResponse{}, fmt.Errorf("GitHub release check returned HTTP %d", resp.StatusCode)
	}
	var release githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubReleaseResponse{}, err
	}
	if release.Draft {
		return githubReleaseResponse{}, fmt.Errorf("latest release is a draft")
	}
	return release, nil
}

func isVersionNewer(latest, current string) bool {
	latestParts := versionParts(latest)
	currentParts := versionParts(current)
	if len(latestParts) == 0 || len(currentParts) == 0 {
		return false
	}
	maxLen := len(latestParts)
	if len(currentParts) > maxLen {
		maxLen = len(currentParts)
	}
	for i := 0; i < maxLen; i++ {
		var l, c int
		if i < len(latestParts) {
			l = latestParts[i]
		}
		if i < len(currentParts) {
			c = currentParts[i]
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}
	return false
}

func versionParts(version string) []int {
	matches := versionPartPattern.FindAllString(version, -1)
	parts := make([]int, 0, len(matches))
	for _, match := range matches {
		part, err := strconv.Atoi(match)
		if err == nil {
			parts = append(parts, part)
		}
	}
	return parts
}

func loadUpdateState() (updateState, error) {
	var state updateState
	data, err := os.ReadFile(updateStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

func saveUpdateState(state updateState) error {
	path := updateStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func updateStatePath() string {
	return filepath.Join(utils.GetJameclawHome(), updateStateFile)
}
