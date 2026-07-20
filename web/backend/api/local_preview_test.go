package api

import "testing"

func TestLocalPreviewTarget(t *testing.T) {
	port, path, ok := localPreviewTarget("/_preview/3000/dashboard")
	if !ok || port != 3000 || path != "/dashboard" {
		t.Fatalf("unexpected parsed preview: port=%d path=%q ok=%v", port, path, ok)
	}
}

func TestLocalPreviewTargetRejectsPrivilegedAndMalformedPorts(t *testing.T) {
	for _, requestPath := range []string{"/_preview/80/", "/_preview/nope/", "/_preview/"} {
		if _, _, ok := localPreviewTarget(requestPath); ok {
			t.Fatalf("expected %q to be rejected", requestPath)
		}
	}
}
