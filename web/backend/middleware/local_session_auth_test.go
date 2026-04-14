package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalSessionAuth_EmptyTokenAllowsAll(t *testing.T) {
	h := LocalSessionAuth("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestLocalSessionAuth_QueryTokenSetsCookieAndRedirects(t *testing.T) {
	h := LocalSessionAuth("secret-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?access_token=secret-token&x=1", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/?x=1" {
		t.Fatalf("Location = %q, want %q", got, "/?x=1")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	if cookies[0].Name != launcherAccessCookieName || cookies[0].Value != "secret-token" {
		t.Fatalf("unexpected cookie = %+v", cookies[0])
	}
	if cookies[1].Name != launcherSplashCookieName || cookies[1].Value != "1" {
		t.Fatalf("unexpected splash cookie = %+v", cookies[1])
	}
}

func TestLocalSessionAuth_CookieAllowsAPI(t *testing.T) {
	h := LocalSessionAuth("secret-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.AddCookie(&http.Cookie{Name: launcherAccessCookieName, Value: "secret-token"})
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestLocalSessionAuth_RejectsAPIWithoutToken(t *testing.T) {
	h := LocalSessionAuth("secret-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"error":"launcher access authentication required"}` {
		t.Fatalf("body = %q", body)
	}
}

func TestLocalSessionAuth_AllowsLocalExtensionBootstrap(t *testing.T) {
	h := LocalSessionAuth("secret-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/extension/bootstrap", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	req.Header.Set("Origin", "chrome-extension://test-extension-id")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
