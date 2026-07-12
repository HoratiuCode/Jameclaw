package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
)

const (
	fileSearchDefaultLimit = 12
	fileSearchMaxLimit     = 25
	fileSearchMaxVisited   = 8000
)

type fileSearchResponse struct {
	Items []fileSearchItem `json:"items"`
}

type fileSearchCache struct {
	RootsKey string
	Items    []fileSearchItem
}

type fileSearchItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Directory  string `json:"directory"`
	Kind       string `json:"kind"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"modified_at_ms"`
}

func (h *Handler) registerFileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/files/search", h.handleFileSearch)
}

func (h *Handler) handleFileSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parsePositiveInt(r.URL.Query().Get("limit"), fileSearchDefaultLimit)
	if limit > fileSearchMaxLimit {
		limit = fileSearchMaxLimit
	}

	roots := h.fileSearchRoots()
	items := searchIndexedLocalFiles(h.cachedLocalFileIndex(roots), query, limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileSearchResponse{Items: items})
}

func (h *Handler) cachedLocalFileIndex(roots []string) []fileSearchItem {
	rootsKey := strings.Join(roots, "\x00")

	h.fileSearchMu.Lock()
	defer h.fileSearchMu.Unlock()

	if h.fileSearchCache.RootsKey == rootsKey && h.fileSearchCache.Items != nil {
		return append([]fileSearchItem(nil), h.fileSearchCache.Items...)
	}

	items := buildLocalFileIndex(roots)
	h.fileSearchCache = fileSearchCache{
		RootsKey: rootsKey,
		Items:    append([]fileSearchItem(nil), items...),
	}
	return items
}

func (h *Handler) fileSearchRoots() []string {
	seen := map[string]bool{}
	var roots []string
	addRoot := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			return
		}
		clean := filepath.Clean(abs)
		if seen[clean] {
			return
		}
		seen[clean] = true
		roots = append(roots, clean)
	}

	if cfg, err := config.LoadConfig(h.configPath); err == nil {
		addRoot(cfg.WorkspacePath())
	}

	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range []string{"Desktop", "Documents", "Downloads"} {
			addRoot(filepath.Join(home, name))
		}
	}

	return roots
}

func searchLocalFiles(roots []string, query string, limit int) []fileSearchItem {
	return searchIndexedLocalFiles(buildLocalFileIndex(roots), query, limit)
}

func buildLocalFileIndex(roots []string) []fileSearchItem {
	var items []fileSearchItem
	visited := 0
	for _, root := range roots {
		if visited >= fileSearchMaxVisited {
			break
		}
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if path == root {
				return nil
			}
			visited++
			if visited > fileSearchMaxVisited {
				return filepath.SkipAll
			}

			name := entry.Name()
			if shouldSkipFileSearchEntry(name, entry.IsDir()) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			info, statErr := entry.Info()
			if statErr != nil {
				return nil
			}
			kind := "file"
			size := info.Size()
			if info.IsDir() {
				kind = "folder"
				size = 0
			}
			items = append(items, fileSearchItem{
				Name:       name,
				Path:       path,
				Directory:  filepath.Dir(path),
				Kind:       kind,
				Size:       size,
				ModifiedAt: info.ModTime().UnixMilli(),
			})
			return nil
		})
	}
	return items
}

func searchIndexedLocalFiles(items []fileSearchItem, query string, limit int) []fileSearchItem {
	if limit <= 0 {
		return nil
	}

	query = strings.ToLower(strings.TrimSpace(query))
	tokens := strings.FieldsFunc(query, func(r rune) bool {
		return r == '/' || r == '\\' || r == '-' || r == '_' || r == '.'
	})

	matches := make([]fileSearchItem, 0, len(items))
	for _, item := range items {
		if query != "" && !fileSearchMatches(item.Path, item.Name, tokens) {
			continue
		}
		matches = append(matches, item)
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Kind != matches[j].Kind {
			return matches[i].Kind == "file"
		}
		return fileSearchRank(matches[i], query) > fileSearchRank(matches[j], query)
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func shouldSkipFileSearchEntry(name string, isDir bool) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	if !isDir {
		return false
	}
	switch name {
	case "node_modules", "dist", "build", "vendor", "target", "__pycache__", "Library":
		return true
	default:
		return false
	}
}

func fileSearchMatches(path string, name string, tokens []string) bool {
	haystack := strings.ToLower(name + " " + path)
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}

func fileSearchRank(item fileSearchItem, query string) int {
	name := strings.ToLower(item.Name)
	path := strings.ToLower(item.Path)
	score := 0
	if query != "" {
		switch {
		case name == query:
			score += 100
		case strings.HasPrefix(name, query):
			score += 75
		case strings.Contains(name, query):
			score += 50
		case strings.Contains(path, query):
			score += 20
		}
	}
	if time.Since(time.UnixMilli(item.ModifiedAt)) < 30*24*time.Hour {
		score += 10
	}
	return score
}
