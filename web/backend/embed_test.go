package main

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestUnknownAPIPathStays404(t *testing.T) {
	mux := http.NewServeMux()
	if err := registerEmbedRoutesWithFS(mux, fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<!doctype html>")},
	}); err != nil {
		t.Fatalf("registerEmbedRoutes() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/not-found", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMissingAssetStays404(t *testing.T) {
	mux := http.NewServeMux()
	if err := registerEmbedRoutesWithFS(mux, fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<!doctype html>")},
	}); err != nil {
		t.Fatalf("registerEmbedRoutes() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/not-found.js", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestEmbeddedFrontendSubFSRequiresIndexHTML(t *testing.T) {
	_, err := embeddedFrontendSubFS(fstest.MapFS{
		"dist/.gitkeep": &fstest.MapFile{Data: []byte("placeholder")},
	})
	if !errors.Is(err, errMissingEmbeddedFrontend) {
		t.Fatalf("expected errMissingEmbeddedFrontend, got %v", err)
	}
}

func TestEmbeddedFrontendSubFSReturnsUsableDist(t *testing.T) {
	subFS, err := embeddedFrontendSubFS(fstest.MapFS{
		"dist/index.html":    &fstest.MapFile{Data: []byte("<!doctype html>")},
		"dist/assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	})
	if err != nil {
		t.Fatalf("embeddedFrontendSubFS() error = %v", err)
	}

	if _, err := fs.Stat(subFS, "index.html"); err != nil {
		t.Fatalf("expected embedded index.html, got %v", err)
	}
}
