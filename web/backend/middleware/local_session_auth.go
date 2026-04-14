package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	launcherAccessCookieName = "jameclaw_launcher_session"
	launcherAccessQueryParam = "access_token"
	launcherSplashCookieName = "jameclaw_shell_launch"
)

// LocalSessionAuth requires a launcher bootstrap token or an established
// session cookie before allowing access to the local web UI and APIs.
func LocalSessionAuth(accessToken string, next http.Handler) http.Handler {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isLocalExtensionRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		if hasLauncherSessionCookie(r, accessToken) {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.Query().Get(launcherAccessQueryParam) == accessToken {
			http.SetCookie(w, &http.Cookie{
				Name:     launcherAccessCookieName,
				Value:    accessToken,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			http.SetCookie(w, &http.Cookie{
				Name:     launcherSplashCookieName,
				Value:    "1",
				Path:     "/",
				MaxAge:   15,
				SameSite: http.SameSiteLaxMode,
			})

			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				redirectURL := stripLauncherAccessToken(r.URL)
				http.Redirect(w, r, redirectURL, http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		rejectUnauthorized(w, r)
	})
}

func isLocalExtensionRequest(r *http.Request) bool {
	if !(strings.HasPrefix(r.URL.Path, "/api/extension/") || strings.HasPrefix(r.URL.Path, "/extension/ws")) {
		return false
	}

	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if !strings.HasPrefix(origin, "chrome-extension://") {
		return false
	}

	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = h
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func hasLauncherSessionCookie(r *http.Request, accessToken string) bool {
	cookie, err := r.Cookie(launcherAccessCookieName)
	return err == nil && cookie.Value == accessToken
}

func stripLauncherAccessToken(u *url.URL) string {
	cleaned := *u
	query := cleaned.Query()
	query.Del(launcherAccessQueryParam)
	cleaned.RawQuery = query.Encode()
	return cleaned.RequestURI()
}

func rejectUnauthorized(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"launcher access authentication required"}`))
		return
	}

	if strings.HasPrefix(r.URL.Path, "/jame/") {
		http.Error(w, "launcher access authentication required", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("launcher access authentication required"))
}
