package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/memory"
	"github.com/sipeed/jameclaw/pkg/providers"
)

// registerSessionRoutes binds session list and detail endpoints to the ServeMux.
func (h *Handler) registerSessionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sessions", h.handleListSessions)
	mux.HandleFunc("PUT /api/sessions/{id}/title", h.handleRenameSession)
	mux.HandleFunc("PUT /api/sessions/{id}/pin", h.handlePinSession)
	mux.HandleFunc("PUT /api/sessions/{id}/archive", h.handleArchiveSession)
	mux.HandleFunc("POST /api/sessions/{id}/resume", h.handleResumeSession)
	mux.HandleFunc("GET /api/sessions/{id}", h.handleGetSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", h.handleDeleteSession)
}

// sessionFile mirrors the on-disk session JSON structure from pkg/session.
type sessionFile struct {
	Key      string              `json:"key"`
	Messages []providers.Message `json:"messages"`
	Summary  string              `json:"summary,omitempty"`
	Created  time.Time           `json:"created"`
	Updated  time.Time           `json:"updated"`
}

// sessionListItem is a lightweight summary returned by GET /api/sessions.
type sessionListItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Preview      string `json:"preview"`
	AgentID      string `json:"agent_id,omitempty"`
	Channel      string `json:"channel,omitempty"`
	ChatType     string `json:"chat_type,omitempty"`
	ChatID       string `json:"chat_id,omitempty"`
	MessageCount int    `json:"message_count"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
	Pinned       bool   `json:"pinned"`
	Archived     bool   `json:"archived"`
}

type sessionRecord struct {
	ID      string
	Session sessionFile
}

type sessionMetaFile struct {
	Key       string    `json:"key"`
	Summary   string    `json:"summary"`
	Skip      int       `json:"skip"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// jameSessionPrefix is the key prefix used by the gateway's routing for Jame
// channel sessions. The full key format is:
//
//	agent:main:jame:direct:jame:<session-uuid>
//
// The sanitized filename replaces ':' with '_', so on disk it becomes:
//
//	agent_main_jame_direct_jame_<session-uuid>.json
const (
	jameSessionPrefix          = "agent:main:jame:direct:jame:"
	agentSessionPrefix         = "agent:"
	sanitizedJameSessionPrefix = "agent_main_jame_direct_jame_"
	maxSessionJSONLLineSize    = 10 * 1024 * 1024 // 10 MB
	maxSessionTitleRunes       = 60
)

// extractJameSessionID extracts the session UUID from a full session key.
// Returns the UUID and true if the key matches the Jame session pattern.
func extractJameSessionID(key string) (string, bool) {
	if strings.HasPrefix(key, jameSessionPrefix) {
		return strings.TrimPrefix(key, jameSessionPrefix), true
	}
	return "", false
}

func extractJameSessionIDFromSanitizedKey(key string) (string, bool) {
	if strings.HasPrefix(key, sanitizedJameSessionPrefix) {
		return strings.TrimPrefix(key, sanitizedJameSessionPrefix), true
	}
	return "", false
}

func sanitizeSessionKey(key string) string {
	key = strings.ReplaceAll(key, ":", "_")
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	return key
}

func sessionIDForKey(key string) string {
	if sessionID, ok := extractJameSessionID(key); ok {
		return sessionID
	}
	return key
}

func sessionKeyForID(id string) string {
	if strings.HasPrefix(id, agentSessionPrefix) {
		return id
	}
	return jameSessionPrefix + id
}

func sessionIdentityForKey(key string) (agentID, channel, chatType, chatID string) {
	parts := strings.SplitN(strings.TrimSpace(key), ":", 3)
	if len(parts) < 3 || parts[0] != "agent" {
		return "", "", "", ""
	}

	agentID = parts[1]
	rest := strings.Split(parts[2], ":")
	if len(rest) == 1 {
		return agentID, rest[0], "", ""
	}

	peerKindIndex := -1
	for i, part := range rest {
		switch part {
		case "direct", "group", "channel":
			peerKindIndex = i
		}
		if peerKindIndex >= 0 {
			break
		}
	}
	if peerKindIndex < 0 {
		return agentID, rest[0], "", ""
	}

	channel = rest[0]
	chatType = rest[peerKindIndex]
	if peerKindIndex+1 < len(rest) {
		chatParts := rest[peerKindIndex+1:]
		if len(chatParts) >= 3 && chatParts[len(chatParts)-2] == "thread" {
			threadID := chatParts[len(chatParts)-1]
			chatID = strings.Join(chatParts[:len(chatParts)-2], ":") + "/thread/" + threadID
		} else {
			chatID = strings.Join(chatParts, ":")
		}
	}
	return agentID, channel, chatType, chatID
}

func (h *Handler) readLegacySession(dir, sessionID string) (sessionFile, error) {
	return h.readLegacySessionByKey(dir, sessionKeyForID(sessionID))
}

func (h *Handler) readLegacySessionByKey(dir, sessionKey string) (sessionFile, error) {
	path := filepath.Join(dir, sanitizeSessionKey(sessionKey)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionFile{}, err
	}

	var sess sessionFile
	if err := json.Unmarshal(data, &sess); err != nil {
		return sessionFile{}, err
	}
	return sess, nil
}

func (h *Handler) readSessionMeta(path, sessionKey string) (sessionMetaFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return sessionMetaFile{Key: sessionKey}, nil
	}
	if err != nil {
		return sessionMetaFile{}, err
	}

	var meta sessionMetaFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return sessionMetaFile{}, err
	}
	if meta.Key == "" {
		meta.Key = sessionKey
	}
	return meta, nil
}

func (h *Handler) readSessionMessages(path string, skip int) ([]providers.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	msgs := make([]providers.Message, 0)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSessionJSONLLineSize)

	seen := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		seen++
		if seen <= skip {
			continue
		}

		var msg providers.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (h *Handler) readJSONLSession(dir, sessionID string) (sessionFile, error) {
	return h.readJSONLSessionByKey(dir, sessionKeyForID(sessionID))
}

func (h *Handler) readJSONLSessionByKey(dir, sessionKey string) (sessionFile, error) {
	base := filepath.Join(dir, sanitizeSessionKey(sessionKey))
	jsonlPath := base + ".jsonl"
	metaPath := base + ".meta.json"

	meta, err := h.readSessionMeta(metaPath, sessionKey)
	if err != nil {
		return sessionFile{}, err
	}

	messages, err := h.readSessionMessages(jsonlPath, meta.Skip)
	if err != nil {
		return sessionFile{}, err
	}

	updated := meta.UpdatedAt
	created := meta.CreatedAt
	if created.IsZero() || updated.IsZero() {
		if info, statErr := os.Stat(jsonlPath); statErr == nil {
			if created.IsZero() {
				created = info.ModTime()
			}
			if updated.IsZero() {
				updated = info.ModTime()
			}
		}
	}

	return sessionFile{
		Key:      meta.Key,
		Messages: messages,
		Summary:  meta.Summary,
		Created:  created,
		Updated:  updated,
	}, nil
}

func buildSessionListItem(sessionID string, sess sessionFile) sessionListItem {
	preview := ""
	for _, msg := range sess.Messages {
		if msg.Role == "user" && strings.TrimSpace(msg.Content) != "" {
			preview = msg.Content
			break
		}
	}
	title := strings.TrimSpace(sess.Summary)
	if title == "" {
		title = preview
	}

	title = truncateRunes(title, maxSessionTitleRunes)
	preview = truncateRunes(preview, maxSessionTitleRunes)

	if preview == "" {
		preview = "(empty)"
	}
	if title == "" {
		title = preview
	}

	validMessageCount := 0
	for _, msg := range sess.Messages {
		if (msg.Role == "user" || msg.Role == "assistant") && strings.TrimSpace(msg.Content) != "" {
			validMessageCount++
		}
	}
	agentID, channel, chatType, chatID := sessionIdentityForKey(sess.Key)

	return sessionListItem{
		ID:           sessionID,
		Title:        title,
		Preview:      preview,
		AgentID:      agentID,
		Channel:      channel,
		ChatType:     chatType,
		ChatID:       chatID,
		MessageCount: validMessageCount,
		Created:      sess.Created.Format(time.RFC3339),
		Updated:      sess.Updated.Format(time.RFC3339),
	}
}

func isEmptySession(sess sessionFile) bool {
	return len(sess.Messages) == 0 && strings.TrimSpace(sess.Summary) == ""
}

func truncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen]) + "..."
}

// sessionsDir resolves the path to the gateway's session storage directory.
// It reads the workspace from config, falling back to ~/.jameclaw/workspace.
func (h *Handler) sessionsDir() (string, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return "", err
	}

	workspace := cfg.Agents.Defaults.Workspace
	if workspace == "" {
		home, _ := os.UserHomeDir()
		workspace = filepath.Join(home, ".jameclaw", "workspace")
	}

	// Expand ~ prefix
	if len(workspace) > 0 && workspace[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(workspace) > 1 && workspace[1] == '/' {
			workspace = home + workspace[1:]
		} else {
			workspace = home
		}
	}

	return filepath.Join(workspace, "sessions"), nil
}

// handleListSessions returns a list of Jame session summaries.
//
//	GET /api/sessions
func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	items := h.listAllSessionItems()
	pinned := h.pinnedSessionIDs()
	archived := h.archivedSessionIDs()
	titles := h.sessionTitles()
	for i := range items {
		items[i].Pinned = pinned[items[i].ID]
		items[i].Archived = archived[items[i].ID]
		if title := strings.TrimSpace(titles[items[i].ID]); title != "" {
			items[i].Title = truncateRunes(title, maxSessionTitleRunes)
		}
	}

	// Pagination parameters
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := 0
	limit := 20 // Default limit

	if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
		offset = val
	}
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		limit = val
	}

	totalItems := len(items)

	end := offset + limit
	if offset >= totalItems {
		items = []sessionListItem{} // Out of bounds, return empty
	} else {
		if end > totalItems {
			end = totalItems
		}
		items = items[offset:end]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *Handler) pinnedSessionsPath() (string, error) {
	dir, err := h.sessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".pinned-chats.json"), nil
}

func (h *Handler) archivedSessionsPath() (string, error) {
	dir, err := h.sessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".archived-chats.json"), nil
}

func (h *Handler) sessionTitlesPath() (string, error) {
	dir, err := h.sessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".session-titles.json"), nil
}

func (h *Handler) sessionTitles() map[string]string {
	path, err := h.sessionTitlesPath()
	if err != nil {
		return map[string]string{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	var titles map[string]string
	if json.Unmarshal(data, &titles) != nil || titles == nil {
		return map[string]string{}
	}
	return titles
}

func (h *Handler) handleRenameSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	title = truncateRunes(title, maxSessionTitleRunes)
	titles := h.sessionTitles()
	titles[id] = title
	path, err := h.sessionTitlesPath()
	if err != nil {
		http.Error(w, "failed to resolve sessions", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, mustJSON(titles), 0600); err != nil {
		http.Error(w, "failed to save session title", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"title": title})
}

func (h *Handler) pinnedSessionIDs() map[string]bool {
	path, err := h.pinnedSessionsPath()
	if err != nil {
		return map[string]bool{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]bool{}
	}
	var ids []string
	if json.Unmarshal(data, &ids) != nil {
		return map[string]bool{}
	}
	result := make(map[string]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func (h *Handler) archivedSessionIDs() map[string]bool {
	path, err := h.archivedSessionsPath()
	if err != nil {
		return map[string]bool{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]bool{}
	}
	var ids []string
	if json.Unmarshal(data, &ids) != nil {
		return map[string]bool{}
	}
	result := make(map[string]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func (h *Handler) handlePinSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	var body struct {
		Pinned bool `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	pinned := h.pinnedSessionIDs()
	if body.Pinned {
		pinned[id] = true
	} else {
		delete(pinned, id)
	}
	ids := make([]string, 0, len(pinned))
	for pinnedID := range pinned {
		ids = append(ids, pinnedID)
	}
	sort.Strings(ids)
	path, err := h.pinnedSessionsPath()
	if err != nil {
		http.Error(w, "failed to resolve sessions", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, mustJSON(ids), 0600); err != nil {
		http.Error(w, "failed to save pinned chats", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"pinned": body.Pinned})
}

func (h *Handler) handleArchiveSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	var body struct {
		Archived bool `json:"archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	archived := h.archivedSessionIDs()
	if body.Archived {
		archived[id] = true
	} else {
		delete(archived, id)
	}
	ids := make([]string, 0, len(archived))
	for archivedID := range archived {
		ids = append(ids, archivedID)
	}
	sort.Strings(ids)
	path, err := h.archivedSessionsPath()
	if err != nil {
		http.Error(w, "failed to resolve sessions", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, mustJSON(ids), 0600); err != nil {
		http.Error(w, "failed to save archived chats", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"archived": body.Archived})
}

func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }

func (h *Handler) listAllSessions() []sessionFile {
	records := h.listAllSessionRecords()
	sessions := make([]sessionFile, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, record.Session)
	}
	return sessions
}

func (h *Handler) listAllSessionRecords() []sessionRecord {
	dir, err := h.sessionsDir()
	if err != nil {
		return []sessionRecord{}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []sessionRecord{}
	}

	records := []sessionRecord{}
	seen := make(map[string]struct{})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		var (
			sessionKey string
			sess       sessionFile
			loadErr    error
		)

		switch {
		case strings.HasSuffix(name, ".jsonl"):
			base := strings.TrimSuffix(name, ".jsonl")
			meta, metaErr := h.readSessionMeta(filepath.Join(dir, base+".meta.json"), "")
			if metaErr != nil {
				continue
			}
			sessionKey = strings.TrimSpace(meta.Key)
			if sessionKey == "" {
				if sessionID, ok := extractJameSessionIDFromSanitizedKey(base); ok {
					sessionKey = jameSessionPrefix + sessionID
				}
			}
			if sessionKey == "" {
				continue
			}
			sess, loadErr = h.readJSONLSessionByKey(dir, sessionKey)
			if loadErr == nil && isEmptySession(sess) {
				continue
			}
		case strings.HasSuffix(name, ".meta.json"):
			continue
		case filepath.Ext(name) == ".json":
			base := strings.TrimSuffix(name, ".json")
			if _, statErr := os.Stat(filepath.Join(dir, base+".jsonl")); statErr == nil {
				if meta, metaErr := h.readSessionMeta(filepath.Join(dir, base+".meta.json"), ""); metaErr == nil {
					jsonlKey := strings.TrimSpace(meta.Key)
					if jsonlKey == "" {
						if jsonlSessionID, found := extractJameSessionIDFromSanitizedKey(base); found {
							jsonlKey = jameSessionPrefix + jsonlSessionID
						}
					}
					if jsonlKey != "" {
						if jsonlSess, jsonlErr := h.readJSONLSessionByKey(dir, jsonlKey); jsonlErr == nil &&
							!isEmptySession(jsonlSess) {
							continue
						}
					}
				}
			}
			if _, statErr := os.Stat(filepath.Join(dir, base+".jsonl")); statErr == nil {
				if jsonlSessionID, found := extractJameSessionIDFromSanitizedKey(base); found {
					if jsonlSess, jsonlErr := h.readJSONLSession(dir, jsonlSessionID); jsonlErr == nil &&
						!isEmptySession(jsonlSess) {
						continue
					}
				}
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &sess); err != nil {
				continue
			}
			if isEmptySession(sess) {
				continue
			}
			sessionKey = strings.TrimSpace(sess.Key)
			if sessionKey == "" {
				continue
			}
			if _, exists := seen[sessionKey]; exists {
				continue
			}
		default:
			continue
		}

		if loadErr != nil {
			continue
		}
		if sessionKey == "" {
			sessionKey = sess.Key
		}
		if sessionKey == "" {
			continue
		}
		if _, exists := seen[sessionKey]; exists {
			continue
		}

		seen[sessionKey] = struct{}{}
		records = append(records, sessionRecord{
			ID:      sessionIDForKey(sessionKey),
			Session: sess,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Session.Updated.After(records[j].Session.Updated)
	})

	return records
}

func (h *Handler) listAllSessionItems() []sessionListItem {
	records := h.listAllSessionRecords()
	items := make([]sessionListItem, 0, len(records))
	for _, record := range records {
		items = append(items, buildSessionListItem(record.ID, record.Session))
	}
	return items
}

// handleGetSession returns the full message history for a specific session.
//
//	GET /api/sessions/{id}
func (h *Handler) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	dir, err := h.sessionsDir()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	sessionKey := sessionKeyForID(sessionID)
	sess, err := h.loadSessionByKey(dir, sessionKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "session not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to parse session", http.StatusInternalServerError)
		}
		return
	}

	// Convert to a simpler format for the frontend
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	messages := make([]chatMessage, 0, len(sess.Messages))
	for _, msg := range sess.Messages {
		// Only include user and assistant messages that have actual content
		if (msg.Role == "user" || msg.Role == "assistant") && strings.TrimSpace(msg.Content) != "" {
			messages = append(messages, chatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       sessionID,
		"messages": messages,
		"summary":  sess.Summary,
		"created":  sess.Created.Format(time.RFC3339),
		"updated":  sess.Updated.Format(time.RFC3339),
	})
}

func (h *Handler) loadSessionByKey(dir, sessionKey string) (sessionFile, error) {
	sess, err := h.readJSONLSessionByKey(dir, sessionKey)
	if err == nil && isEmptySession(sess) {
		err = os.ErrNotExist
	}
	if errors.Is(err, os.ErrNotExist) {
		sess, err = h.readLegacySessionByKey(dir, sessionKey)
		if err == nil && isEmptySession(sess) {
			err = os.ErrNotExist
		}
	}
	return sess, err
}

// handleResumeSession returns a Jame-compatible session ID and its visible
// transcript. Jame sessions keep their original ID. Sessions created through
// Telegram, Discord, or another channel are copied into a new Jame session so
// the native WebSocket can safely continue them without taking over the
// original channel's routing identity.
//
//	POST /api/sessions/{id}/resume
func (h *Handler) handleResumeSession(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.PathValue("id"))
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	dir, err := h.sessionsDir()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	sourceKey := sessionKeyForID(sessionID)
	sess, err := h.loadSessionByKey(dir, sourceKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "session not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to parse session", http.StatusInternalServerError)
		}
		return
	}

	resumeID, isJameSession := extractJameSessionID(sourceKey)
	if !isJameSession {
		resumeID = uuid.NewString()
		targetKey := jameSessionPrefix + resumeID
		store, storeErr := memory.NewJSONLStore(dir)
		if storeErr != nil {
			http.Error(w, "failed to prepare resumable session", http.StatusInternalServerError)
			return
		}
		for _, msg := range sess.Messages {
			if addErr := store.AddFullMessage(r.Context(), targetKey, msg); addErr != nil {
				http.Error(w, "failed to copy session history", http.StatusInternalServerError)
				return
			}
		}
		if strings.TrimSpace(sess.Summary) != "" {
			if summaryErr := store.SetSummary(r.Context(), targetKey, sess.Summary); summaryErr != nil {
				http.Error(w, "failed to copy session summary", http.StatusInternalServerError)
				return
			}
		}
	}

	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]chatMessage, 0, len(sess.Messages))
	for _, msg := range sess.Messages {
		if (msg.Role == "user" || msg.Role == "assistant") && strings.TrimSpace(msg.Content) != "" {
			messages = append(messages, chatMessage{Role: msg.Role, Content: msg.Content})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"session_id": resumeID,
		"messages":   messages,
		"cloned":     !isJameSession,
	})
}

// handleDeleteSession deletes a specific session.
//
//	DELETE /api/sessions/{id}
func (h *Handler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	dir, err := h.sessionsDir()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	sessionKey := sessionKeyForID(sessionID)
	base := filepath.Join(dir, sanitizeSessionKey(sessionKey))
	jsonlPath := base + ".jsonl"
	metaPath := base + ".meta.json"
	legacyPath := base + ".json"

	removed := false
	for _, path := range []string{jsonlPath, metaPath, legacyPath} {
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			http.Error(w, "failed to delete session", http.StatusInternalServerError)
			return
		}
		removed = true
	}

	if !removed {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
