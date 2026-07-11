package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func withUpdateHTTPClient(t *testing.T, fn roundTripFunc) {
	t.Helper()
	oldClient := githubReleaseHTTPClient
	githubReleaseHTTPClient = &http.Client{Transport: fn}
	t.Cleanup(func() { githubReleaseHTTPClient = oldClient })
}

func TestGetUpdateStatus_NoPublishedGitHubReleaseIsNotCheckError(t *testing.T) {
	withUpdateHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != githubLatestReleaseURL {
			t.Fatalf("request URL = %q, want %q", req.URL.String(), githubLatestReleaseURL)
		}
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"message":"Not Found"}`)),
			Header:     make(http.Header),
		}, nil
	})

	oldVersion := config.Version
	config.Version = "121de420-dirty"
	t.Cleanup(func() { config.Version = oldVersion })

	status := NewHandler("").getUpdateStatus(t.Context())
	if status.CheckError != "" {
		t.Fatalf("CheckError = %q, want empty", status.CheckError)
	}
	if status.UpdateAvailable {
		t.Fatal("UpdateAvailable = true, want false")
	}
	if status.LatestVersion != "" {
		t.Fatalf("LatestVersion = %q, want empty", status.LatestVersion)
	}
}

func TestGetUpdateStatus_UsesLatestRelease(t *testing.T) {
	withUpdateHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"tag_name": "v2.0.0",
				"name": "JameClaw v2.0.0",
				"html_url": "https://github.com/HoratiuCode/Jameclaw/releases/tag/v2.0.0",
				"published_at": "2026-07-10T12:00:00Z"
			}`)),
			Header: make(http.Header),
		}, nil
	})

	oldVersion := config.Version
	config.Version = "v1.0.0"
	t.Cleanup(func() { config.Version = oldVersion })

	status := NewHandler("").getUpdateStatus(t.Context())
	if status.CheckError != "" {
		t.Fatalf("CheckError = %q, want empty", status.CheckError)
	}
	if !status.UpdateAvailable {
		t.Fatal("UpdateAvailable = false, want true")
	}
	if status.LatestVersion != "v2.0.0" {
		t.Fatalf("LatestVersion = %q, want v2.0.0", status.LatestVersion)
	}
	if status.ReleaseURL != "https://github.com/HoratiuCode/Jameclaw/releases/tag/v2.0.0" {
		t.Fatalf("ReleaseURL = %q", status.ReleaseURL)
	}
}
