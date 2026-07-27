package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
)

// An artifact is a small self-contained project made by the agent or edited in
// the Web Console.  Files live under the configured workspace so they remain
// available to the CLI and to future conversations.
type artifact struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	HTML       string `json:"html"`
	CSS        string `json:"css"`
	JavaScript string `json:"javascript"`
	CreatedAt  int64  `json:"created_at_ms"`
	UpdatedAt  int64  `json:"updated_at_ms"`
}

type artifactListResponse struct {
	Items []artifact `json:"items"`
}

type artifactFile struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Content  string `json:"content"`
	Size     int    `json:"size"`
}

type artifactFilesResponse struct {
	Folder string         `json:"folder"`
	Files  []artifactFile `json:"files"`
}

func (h *Handler) registerArtifactRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/artifacts", h.handleArtifactList)
	mux.HandleFunc("POST /api/artifacts", h.handleArtifactCreate)
	mux.HandleFunc("GET /api/artifacts/{id}", h.handleArtifactGet)
	mux.HandleFunc("GET /api/artifacts/{id}/files", h.handleArtifactFiles)
	mux.HandleFunc("PUT /api/artifacts/{id}", h.handleArtifactUpdate)
	mux.HandleFunc("DELETE /api/artifacts/{id}", h.handleArtifactDelete)
	mux.HandleFunc("GET /api/artifacts/{id}/preview", h.handleArtifactPreview)
}

func (h *Handler) artifactsDir() (string, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg.WorkspacePath(), "artifacts"), nil
}

func artifactID() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func artifactPath(root, id string) (string, bool) {
	if len(id) != 20 {
		return "", false
	}
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return "", false
		}
	}
	return filepath.Join(root, id, "artifact.json"), true
}

func readArtifact(root, id string) (artifact, error) {
	path, ok := artifactPath(root, id)
	if !ok {
		return artifact{}, os.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return artifact{}, err
	}
	var item artifact
	if err := json.Unmarshal(data, &item); err != nil {
		return artifact{}, err
	}
	return item, nil
}

func writeArtifact(root string, item artifact) error {
	path, ok := artifactPath(root, item.ID)
	if !ok {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	for _, file := range artifactFiles(item) {
		if err := os.WriteFile(filepath.Join(filepath.Dir(path), file.Name), []byte(file.Content), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func artifactFiles(item artifact) []artifactFile {
	index := "<!doctype html>\n<html>\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>" + item.Name + "</title>\n  <link rel=\"stylesheet\" href=\"styles.css\">\n</head>\n<body>\n" + item.HTML + "\n  <script src=\"script.js\"></script>\n</body>\n</html>\n"
	files := []artifactFile{
		{Name: "index.html", Language: "html", Content: index},
		{Name: "styles.css", Language: "css", Content: item.CSS},
		{Name: "script.js", Language: "javascript", Content: item.JavaScript},
	}
	for i := range files {
		files[i].Size = len([]byte(files[i].Content))
	}
	return files
}

func decodeArtifact(w http.ResponseWriter, r *http.Request) (artifact, bool) {
	var item artifact
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&item); err != nil {
		http.Error(w, "invalid artifact payload", http.StatusBadRequest)
		return artifact{}, false
	}
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		http.Error(w, "artifact name is required", http.StatusBadRequest)
		return artifact{}, false
	}
	if len(item.Name) > 120 {
		http.Error(w, "artifact name is too long", http.StatusBadRequest)
		return artifact{}, false
	}
	if item.Kind == "" {
		item.Kind = "app"
	}
	if item.Kind != "app" && item.Kind != "code" {
		http.Error(w, "invalid artifact kind", http.StatusBadRequest)
		return artifact{}, false
	}
	return item, true
}

func (h *Handler) handleArtifactList(w http.ResponseWriter, r *http.Request) {
	root, err := h.artifactsDir()
	if err != nil {
		http.Error(w, "failed to load workspace", http.StatusInternalServerError)
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, "failed to list artifacts", http.StatusInternalServerError)
		return
	}
	items := make([]artifact, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		item, err := readArtifact(root, entry.Name())
		if err == nil {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artifactListResponse{Items: items})
}

func (h *Handler) handleArtifactCreate(w http.ResponseWriter, r *http.Request) {
	item, ok := decodeArtifact(w, r)
	if !ok {
		return
	}
	root, err := h.artifactsDir()
	if err != nil {
		http.Error(w, "failed to load workspace", http.StatusInternalServerError)
		return
	}
	id, err := artifactID()
	if err != nil {
		http.Error(w, "failed to create artifact", http.StatusInternalServerError)
		return
	}
	now := time.Now().UnixMilli()
	item.ID, item.CreatedAt, item.UpdatedAt = id, now, now
	if err := writeArtifact(root, item); err != nil {
		http.Error(w, "failed to save artifact", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *Handler) handleArtifactGet(w http.ResponseWriter, r *http.Request) {
	root, err := h.artifactsDir()
	if err != nil {
		http.Error(w, "failed to load workspace", http.StatusInternalServerError)
		return
	}
	item, err := readArtifact(root, r.PathValue("id"))
	if err != nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *Handler) handleArtifactFiles(w http.ResponseWriter, r *http.Request) {
	root, err := h.artifactsDir()
	if err != nil {
		http.Error(w, "failed to load workspace", http.StatusInternalServerError)
		return
	}
	item, err := readArtifact(root, r.PathValue("id"))
	if err != nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	files := artifactFiles(item)
	// Read the real project files when present. Older artifacts are upgraded on
	// their next save but remain visible immediately through their metadata.
	path, _ := artifactPath(root, item.ID)
	for i := range files {
		if content, readErr := os.ReadFile(filepath.Join(filepath.Dir(path), files[i].Name)); readErr == nil {
			files[i].Content = string(content)
			files[i].Size = len(content)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artifactFilesResponse{Folder: filepath.Join("artifacts", item.ID), Files: files})
}

func (h *Handler) handleArtifactUpdate(w http.ResponseWriter, r *http.Request) {
	root, err := h.artifactsDir()
	if err != nil {
		http.Error(w, "failed to load workspace", http.StatusInternalServerError)
		return
	}
	existing, err := readArtifact(root, r.PathValue("id"))
	if err != nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	item, ok := decodeArtifact(w, r)
	if !ok {
		return
	}
	item.ID, item.CreatedAt, item.UpdatedAt = existing.ID, existing.CreatedAt, time.Now().UnixMilli()
	if err := writeArtifact(root, item); err != nil {
		http.Error(w, "failed to save artifact", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *Handler) handleArtifactDelete(w http.ResponseWriter, r *http.Request) {
	root, err := h.artifactsDir()
	if err != nil {
		http.Error(w, "failed to load workspace", http.StatusInternalServerError)
		return
	}
	path, ok := artifactPath(root, r.PathValue("id"))
	if !ok {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	if err := os.RemoveAll(filepath.Dir(path)); err != nil {
		http.Error(w, "failed to delete artifact", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleArtifactPreview(w http.ResponseWriter, r *http.Request) {
	root, err := h.artifactsDir()
	if err != nil {
		http.Error(w, "failed to load workspace", http.StatusInternalServerError)
		return
	}
	item, err := readArtifact(root, r.PathValue("id"))
	if err != nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	if item.Kind != "app" {
		http.Error(w, "code artifacts do not have a preview", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// The iframe is sandboxed by the UI. Keeping the composition server-side
	// makes this a normal runnable single-page website without exposing paths.
	io.WriteString(w, "<!doctype html><html><head><meta charset=\"utf-8\"><style>"+item.CSS+"</style></head><body>"+item.HTML+"<script>"+item.JavaScript+"</script></body></html>")
}
