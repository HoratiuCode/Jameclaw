package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArtifactCreateUpdateAndPreview(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	payload := []byte(`{"name":"Counter","kind":"app","html":"<button>Count</button>","css":"button{color:red}","javascript":""}`)
	create := httptest.NewRecorder()
	mux.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/artifacts", bytes.NewReader(payload)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", create.Code, create.Body.String())
	}
	var item artifact
	if err := json.Unmarshal(create.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID == "" || item.CreatedAt == 0 {
		t.Fatalf("created artifact = %#v", item)
	}

	preview := httptest.NewRecorder()
	mux.ServeHTTP(preview, httptest.NewRequest(http.MethodGet, "/api/artifacts/"+item.ID+"/preview", nil))
	if preview.Code != http.StatusOK || !bytes.Contains(preview.Body.Bytes(), []byte("Count")) {
		t.Fatalf("preview status = %d, body=%s", preview.Code, preview.Body.String())
	}

	files := httptest.NewRecorder()
	mux.ServeHTTP(files, httptest.NewRequest(http.MethodGet, "/api/artifacts/"+item.ID+"/files", nil))
	if files.Code != http.StatusOK || !bytes.Contains(files.Body.Bytes(), []byte("index.html")) {
		t.Fatalf("files status = %d, body=%s", files.Code, files.Body.String())
	}

	updated := []byte(`{"name":"Counter v2","kind":"code","html":"","css":"","javascript":"const count = 2"}`)
	update := httptest.NewRecorder()
	mux.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/api/artifacts/"+item.ID, bytes.NewReader(updated)))
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d, body=%s", update.Code, update.Body.String())
	}

	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/artifacts", nil))
	var response artifactListResponse
	if err := json.Unmarshal(list.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || response.Items[0].Name != "Counter v2" || response.Items[0].Kind != "code" {
		t.Fatalf("listed artifacts = %#v", response.Items)
	}
}
