package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

const localPreviewPrefix = "/_preview/"

// registerLocalPreviewRoutes exposes explicitly requested loopback HTTP
// previews through the authenticated JameClaw console. It is deliberately
// limited to unprivileged localhost ports and removes credentials so a local
// development app cannot inherit access to the console.
func (h *Handler) registerLocalPreviewRoutes(mux *http.ServeMux) {
	mux.HandleFunc(localPreviewPrefix, h.handleLocalPreview)
}

func (h *Handler) handleLocalPreview(w http.ResponseWriter, r *http.Request) {
	port, previewPath, ok := localPreviewTarget(r.URL.Path)
	if !ok {
		http.Error(w, "A local preview URL must use /_preview/{port}/...", http.StatusBadRequest)
		return
	}

	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = previewPath
		req.URL.RawPath = ""
		req.Host = target.Host
		// Do not forward console credentials, or let the local preview use them.
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
		req.Header.Del("Origin")
		req.Header.Del("Referer")
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Set-Cookie")
		resp.Header.Del("X-Frame-Options")
		resp.Header.Set("Referrer-Policy", "no-referrer")
		// Give preview documents a unique opaque origin. This lets scripts and
		// hot-reload clients run, while preventing generated code from reading
		// the console's API or session cookies.
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			resp.Header.Set("Content-Security-Policy", "sandbox allow-forms allow-modals allow-popups allow-scripts")
		}
		if location := resp.Header.Get("Location"); strings.HasPrefix(location, "/") {
			resp.Header.Set("Location", fmt.Sprintf("%s%d%s", localPreviewPrefix, port, location))
		}
		return nil
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(rw, fmt.Sprintf("Nothing is listening on localhost:%d yet. Start the app, then open this preview again.", port), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func localPreviewTarget(requestPath string) (port int, previewPath string, ok bool) {
	rest := strings.TrimPrefix(requestPath, localPreviewPrefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return 0, "", false
	}
	port, err := strconv.Atoi(parts[0])
	if err != nil || port < 1024 || port > 65535 {
		return 0, "", false
	}
	previewPath = "/"
	if len(parts) == 2 {
		previewPath += parts[1]
	}
	return port, previewPath, true
}
