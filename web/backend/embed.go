package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/sipeed/jameclaw/pkg/logger"
)

//go:embed all:dist
var frontendFS embed.FS

var errMissingEmbeddedFrontend = errors.New("embedded frontend build is missing")

// registerEmbedRoutes sets up the HTTP handler to serve the embedded frontend files
func registerEmbedRoutes(mux *http.ServeMux) error {
	return registerEmbedRoutesWithFS(mux, frontendFS)
}

func registerEmbedRoutesWithFS(mux *http.ServeMux, frontend fs.FS) error {
	// Register correct MIME type for SVG files
	// Go's built-in mime.TypeByExtension returns "image/svg" which is incorrect
	// The correct MIME type per RFC 6838 is "image/svg+xml"
	if err := mime.AddExtensionType(".svg", "image/svg+xml"); err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: failed to register SVG MIME type: %v", err))
	}

	subFS, err := embeddedFrontendSubFS(frontend)
	if err != nil {
		return err
	}

	fileServer := http.FileServer(http.FS(subFS))

	// Serve static assets and fallback to index.html for SPA routes.
	mux.Handle(
		"/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.NotFound(w, r)
				return
			}

			// Keep unknown API paths as 404 instead of falling back to SPA entry.
			if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			cleanPath := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
			if cleanPath == "." {
				cleanPath = ""
			}

			// Existing static files/directories should be served directly.
			if cleanPath != "" {
				if _, statErr := fs.Stat(subFS, cleanPath); statErr == nil {
					fileServer.ServeHTTP(w, r)
					return
				}
				// Missing asset-like paths should remain 404.
				if strings.Contains(path.Base(cleanPath), ".") {
					fileServer.ServeHTTP(w, r)
					return
				}
			}

			indexReq := r.Clone(r.Context())
			indexReq.URL.Path = "/"
			fileServer.ServeHTTP(w, indexReq)
		}),
	)

	return nil
}

func embeddedFrontendSubFS(frontend fs.FS) (fs.FS, error) {
	subFS, err := fs.Sub(frontend, "dist")
	if err != nil {
		return nil, fmt.Errorf(
			"%w: embedded dist directory not found; run `pnpm build:backend` in `web/frontend` before building `jameclaw-web`",
			errMissingEmbeddedFrontend,
		)
	}

	if _, err := fs.Stat(subFS, "index.html"); err != nil {
		return nil, fmt.Errorf(
			"%w: embedded dist/index.html not found; run `pnpm build:backend` in `web/frontend` before building `jameclaw-web`",
			errMissingEmbeddedFrontend,
		)
	}

	return subFS, nil
}
