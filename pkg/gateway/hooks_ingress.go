package gateway

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/logger"
)

const (
	defaultHookIngressPath         = "/hooks"
	defaultHookIngressBodyBytes    = 256 * 1024
	defaultHookAgentTimeoutSeconds = 120
	defaultHookWakeTimeoutSeconds  = 30
	hookAuthFailureLimit           = 20
	hookAuthFailureWindow          = 60 * time.Second
	hookTokenHeader                = "X-JameClaw-Token"
	hookResponseContentType        = "application/json; charset=utf-8"
	hookSessionPrefix              = "hook:"
	hookDeliveredDefaultChannel    = "cli"
	hookInternalChatID             = "internal"
)

type hookRunner interface {
	ProcessHeartbeat(ctx context.Context, content, channel, chatID string) (string, error)
	ProcessDirectOnAgent(
		ctx context.Context,
		agentID, content, sessionKey, channel, chatID string,
	) (string, error)
	GetLastChannel() string
	GetLastChatID() string
}

type hookIngressResolvedConfig struct {
	basePath                  string
	token                     string
	maxBodyBytes              int64
	defaultSessionKey         string
	allowRequestSessionKey    bool
	allowedSessionKeyPrefixes []string
	allowedAgentIDs           map[string]struct{}
	allowAnyAgentID           bool
}

type hookIngressServer struct {
	cfg         hookIngressResolvedConfig
	runner      hookRunner
	publish     func(context.Context, bus.OutboundMessage) error
	authLimiter *hookAuthLimiter
	mappings    []hookMappingResolved
}

type hookAuthLimiter struct {
	mu      sync.Mutex
	entries map[string]*hookAuthLimitEntry
}

type hookAuthLimitEntry struct {
	attempts    []time.Time
	lockedUntil time.Time
}

func newHookAuthLimiter() *hookAuthLimiter {
	return &hookAuthLimiter{
		entries: make(map[string]*hookAuthLimitEntry),
	}
}

func (l *hookAuthLimiter) Check(ip string) (bool, time.Duration) {
	if l == nil || ip == "" {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entryFor(ip)
	now := time.Now()
	if now.Before(entry.lockedUntil) {
		return false, time.Until(entry.lockedUntil)
	}
	entry.attempts = pruneHookAttempts(entry.attempts, now.Add(-hookAuthFailureWindow))
	if len(entry.attempts) >= hookAuthFailureLimit {
		entry.lockedUntil = now.Add(hookAuthFailureWindow)
		return false, time.Until(entry.lockedUntil)
	}
	return true, 0
}

func (l *hookAuthLimiter) RecordFailure(ip string) {
	if l == nil || ip == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entryFor(ip)
	now := time.Now()
	entry.attempts = pruneHookAttempts(entry.attempts, now.Add(-hookAuthFailureWindow))
	entry.attempts = append(entry.attempts, now)
	if len(entry.attempts) >= hookAuthFailureLimit {
		entry.lockedUntil = now.Add(hookAuthFailureWindow)
	}
}

func (l *hookAuthLimiter) Reset(ip string) {
	if l == nil || ip == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

func (l *hookAuthLimiter) entryFor(ip string) *hookAuthLimitEntry {
	if entry, ok := l.entries[ip]; ok {
		return entry
	}
	entry := &hookAuthLimitEntry{}
	l.entries[ip] = entry
	return entry
}

func pruneHookAttempts(attempts []time.Time, cutoff time.Time) []time.Time {
	keep := attempts[:0]
	for _, ts := range attempts {
		if ts.After(cutoff) {
			keep = append(keep, ts)
		}
	}
	return keep
}

func newHookIngressServer(
	cfg *config.Config,
	configDir string,
	runner hookRunner,
	publish func(context.Context, bus.OutboundMessage) error,
) (*hookIngressServer, error) {
	if cfg == nil {
		return nil, nil
	}
	raw := cfg.Hooks.Ingress
	if !raw.Enabled {
		return nil, nil
	}

	token := strings.TrimSpace(raw.Token)
	if token == "" {
		return nil, fmt.Errorf("hooks.ingress.enabled requires hooks.ingress.token")
	}

	basePath := strings.TrimSpace(raw.Path)
	if basePath == "" {
		basePath = defaultHookIngressPath
	}
	basePath = normalizeHookPath(basePath)
	if basePath == "/" {
		return nil, fmt.Errorf("hooks.ingress.path may not be '/'")
	}

	maxBodyBytes := raw.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultHookIngressBodyBytes
	}

	allowedAgentIDs, allowAnyAgentID := resolveAllowedAgentIDs(raw.AllowedAgentIds)
	allowedSessionPrefixes := resolveAllowedSessionKeyPrefixes(raw.AllowedSessionKeyPrefixes)
	defaultSessionKey := strings.TrimSpace(raw.DefaultSessionKey)
	if defaultSessionKey != "" && len(allowedSessionPrefixes) > 0 &&
		!isSessionKeyAllowedByPrefix(defaultSessionKey, allowedSessionPrefixes) {
		return nil, fmt.Errorf("hooks.ingress.default_session_key must match hooks.ingress.allowed_session_key_prefixes")
	}
	if defaultSessionKey == "" && len(allowedSessionPrefixes) > 0 &&
		!isSessionKeyAllowedByPrefix(hookSessionPrefix+"example", allowedSessionPrefixes) {
		return nil, fmt.Errorf(
			"hooks.ingress.allowed_session_key_prefixes must include %q when hooks.ingress.default_session_key is unset",
			hookSessionPrefix,
		)
	}

	mappings, err := resolveHookMappings(cfg.Hooks, configDir)
	if err != nil {
		return nil, err
	}

	return &hookIngressServer{
		cfg: hookIngressResolvedConfig{
			basePath:                  basePath,
			token:                     token,
			maxBodyBytes:              int64(maxBodyBytes),
			defaultSessionKey:         defaultSessionKey,
			allowRequestSessionKey:    raw.AllowRequestSessionKey,
			allowedSessionKeyPrefixes: allowedSessionPrefixes,
			allowedAgentIDs:           allowedAgentIDs,
			allowAnyAgentID:           allowAnyAgentID,
		},
		runner:      runner,
		publish:     publish,
		authLimiter: newHookAuthLimiter(),
		mappings:    mappings,
	}, nil
}

func createHookIngressRegistrar(
	cfg *config.Config,
	configDir string,
	runner hookRunner,
	publish func(context.Context, bus.OutboundMessage) error,
) func(*http.ServeMux) {
	server, err := newHookIngressServer(cfg, configDir, runner, publish)
	if err != nil {
		logger.ErrorCF("gateway", "Failed to configure hook ingress", map[string]any{
			"error": err.Error(),
		})
		return nil
	}
	if server == nil {
		return nil
	}
	return func(mux *http.ServeMux) {
		mux.Handle(server.cfg.basePath, server)
		mux.Handle(server.cfg.basePath+"/", server)
		logger.InfoCF("gateway", "Hook ingress registered", map[string]any{
			"path": server.cfg.basePath,
		})
	}
}

func (s *hookIngressServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		http.NotFound(w, r)
		return
	}

	if strings.Contains(r.URL.RawQuery, "token=") {
		writeHookJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "hook token must be provided via Authorization: Bearer <token> or X-JameClaw-Token header",
		})
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeHookText(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	if !s.authorizeRequest(w, r) {
		return
	}

	payload, err := readHookJSONBody(w, r, s.cfg.maxBodyBytes)
	if err != nil {
		writeHookJSON(w, statusForHookBodyError(err), map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	switch resolveHookSubPath(s.cfg.basePath, r.URL.Path) {
	case "wake":
		s.handleWake(w, r, payload)
	case "agent":
		s.handleAgent(w, r, payload)
	default:
		if !s.handleMappedHook(w, r, payload) {
			http.NotFound(w, r)
		}
	}
}

func (s *hookIngressServer) authorizeRequest(w http.ResponseWriter, r *http.Request) bool {
	clientKey := resolveHookClientKey(r)
	token := extractHookToken(r)
	if !hookSecretsEqual(token, s.cfg.token) {
		allowed, retryAfter := s.authLimiter.Check(clientKey)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(retryAfter.Seconds()))))
			writeHookText(w, http.StatusTooManyRequests, "Too Many Requests")
			logger.WarnCF("gateway", "Hook auth throttled", map[string]any{
				"client": clientKey,
			})
			return false
		}
		s.authLimiter.RecordFailure(clientKey)
		writeHookText(w, http.StatusUnauthorized, "Unauthorized")
		return false
	}

	s.authLimiter.Reset(clientKey)
	return true
}

func (s *hookIngressServer) handleWake(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	s.handleWakeAction(w, r, hookAction{
		kind:           hookMappingActionWake,
		text:           stringField(payload["text"]),
		mode:           normalizeHookWakeMode(stringField(payload["mode"])),
		timeoutSeconds: intField(payload["timeoutSeconds"], defaultHookWakeTimeoutSeconds),
	})
}

func (s *hookIngressServer) handleAgent(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	s.handleAgentAction(w, r, hookAction{
		kind:           hookMappingActionAgent,
		message:        stringField(payload["message"]),
		name:           stringField(payload["name"]),
		agentID:        stringField(payload["agentId"]),
		sessionKey:     stringField(payload["sessionKey"]),
		deliver:        boolPtr(boolField(payload["deliver"], true)),
		channel:        stringField(payload["channel"]),
		to:             stringField(payload["to"]),
		timeoutSeconds: intField(payload["timeoutSeconds"], defaultHookAgentTimeoutSeconds),
	})
}

func (s *hookIngressServer) handleMappedHook(w http.ResponseWriter, r *http.Request, payload map[string]any) bool {
	outcome, err := applyHookMappings(s.mappings, hookTemplateContext{
		Payload: payload,
		Headers: normalizeHookHeaders(r.Header),
		URL:     r.URL.String(),
		Path:    resolveHookSubPath(s.cfg.basePath, r.URL.Path),
	})
	if err != nil {
		writeHookJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return true
	}
	if !outcome.Matched {
		return false
	}
	if outcome.Skipped {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	if outcome.Action == nil {
		return false
	}

	switch outcome.Action.kind {
	case hookMappingActionWake:
		s.handleWakeAction(w, r, *outcome.Action)
	case hookMappingActionAgent:
		s.handleAgentAction(w, r, *outcome.Action)
	default:
		writeHookJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "unsupported hook action",
		})
	}
	return true
}

func (s *hookIngressServer) handleWakeAction(w http.ResponseWriter, r *http.Request, action hookAction) {
	text := strings.TrimSpace(action.text)
	if text == "" {
		writeHookJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "text required",
		})
		return
	}

	mode := normalizeHookWakeMode(action.mode)
	timeoutSeconds := action.timeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultHookWakeTimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	response, err := s.runner.ProcessHeartbeat(ctx, text, "system", "hooks:wake")
	if err != nil {
		writeHookJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	resp := map[string]any{
		"ok":   true,
		"mode": mode,
	}
	if strings.TrimSpace(response) != "" {
		resp["result"] = response
	}
	writeHookJSON(w, http.StatusOK, resp)
}

func (s *hookIngressServer) handleAgentAction(w http.ResponseWriter, r *http.Request, action hookAction) {
	message := strings.TrimSpace(action.message)
	if message == "" {
		writeHookJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "message required",
		})
		return
	}

	name := strings.TrimSpace(action.name)
	if name == "" {
		name = "Hook"
	}

	agentID := strings.TrimSpace(action.agentID)
	if !s.isAgentAllowed(agentID) {
		writeHookJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "agentId is not allowed by hooks.ingress.allowed_agent_ids",
		})
		return
	}

	sessionKey, err := s.resolveConfiguredSessionKey(action.sessionKey)
	if err != nil {
		writeHookJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	deliver := boolValue(action.deliver, true)
	channel, chatID, err := s.resolveDeliveryTarget(deliver, action.channel, action.to)
	if err != nil {
		writeHookJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	timeoutSeconds := action.timeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultHookAgentTimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	result, err := s.runner.ProcessDirectOnAgent(ctx, agentID, message, sessionKey, channel, chatID)
	if err != nil {
		writeHookJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	if deliver && strings.TrimSpace(result) != "" && s.publish != nil {
		pubErr := s.publish(ctx, bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: result,
		})
		if pubErr != nil {
			writeHookJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": pubErr.Error(),
			})
			return
		}
	}

	resp := map[string]any{
		"ok":          true,
		"name":        name,
		"session_key": sessionKey,
		"agent_id":    agentID,
		"delivered":   deliver,
	}
	if deliver {
		resp["channel"] = channel
		resp["to"] = chatID
	}
	if strings.TrimSpace(result) != "" {
		resp["result"] = result
	}
	writeHookJSON(w, http.StatusOK, resp)
}

func (s *hookIngressServer) isAgentAllowed(agentID string) bool {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" || s.cfg.allowAnyAgentID {
		return true
	}
	_, ok := s.cfg.allowedAgentIDs[agentID]
	return ok
}

func (s *hookIngressServer) resolveSessionKey(payload map[string]any) (string, error) {
	requested := stringField(payload["sessionKey"])
	if requested != "" {
		if !s.cfg.allowRequestSessionKey {
			return "", fmt.Errorf("sessionKey is disabled for external hook payloads")
		}
	}
	return s.resolveConfiguredSessionKey(requested)
}

func (s *hookIngressServer) resolveConfiguredSessionKey(requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		if len(s.cfg.allowedSessionKeyPrefixes) > 0 && !isSessionKeyAllowedByPrefix(requested, s.cfg.allowedSessionKeyPrefixes) {
			return "", fmt.Errorf("sessionKey must start with one of: %s", strings.Join(s.cfg.allowedSessionKeyPrefixes, ", "))
		}
		return requested, nil
	}

	if s.cfg.defaultSessionKey != "" {
		return s.cfg.defaultSessionKey, nil
	}

	sessionKey := newHookSessionKey()
	if len(s.cfg.allowedSessionKeyPrefixes) > 0 && !isSessionKeyAllowedByPrefix(sessionKey, s.cfg.allowedSessionKeyPrefixes) {
		return "", fmt.Errorf("generated sessionKey must start with one of: %s", strings.Join(s.cfg.allowedSessionKeyPrefixes, ", "))
	}
	return sessionKey, nil
}

func (s *hookIngressServer) resolveDeliveryTarget(deliver bool, channel, to string) (string, string, error) {
	if !deliver {
		return hookDeliveredDefaultChannel, hookInternalChatID, nil
	}

	channel = strings.TrimSpace(channel)
	to = strings.TrimSpace(to)
	if channel == "" || strings.EqualFold(channel, "last") {
		channel = strings.TrimSpace(s.runner.GetLastChannel())
		if channel == "" {
			return "", "", fmt.Errorf("deliver=true requires a channel or a recorded last channel")
		}
	}
	if to == "" {
		to = strings.TrimSpace(s.runner.GetLastChatID())
		if to == "" {
			return "", "", fmt.Errorf("deliver=true requires to or a recorded last chat id")
		}
	}
	return channel, to, nil
}

func extractHookToken(req *http.Request) string {
	auth := strings.TrimSpace(req.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[7:])
		if token != "" {
			return token
		}
	}
	token := strings.TrimSpace(req.Header.Get(hookTokenHeader))
	if token != "" {
		return token
	}
	return ""
}

func readHookJSONBody(w http.ResponseWriter, req *http.Request, maxBytes int64) (map[string]any, error) {
	req.Body = http.MaxBytesReader(w, req.Body, maxBytes)
	defer req.Body.Close()

	dec := json.NewDecoder(req.Body)
	dec.UseNumber()

	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			return nil, fmt.Errorf("payload too large")
		}
		if err == io.EOF {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("invalid request body")
	}

	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func statusForHookBodyError(err error) int {
	switch err.Error() {
	case "payload too large":
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusBadRequest
	}
}

func writeHookJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", hookResponseContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeHookText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func resolveHookSubPath(basePath, requestPath string) string {
	if !strings.HasPrefix(requestPath, basePath) {
		return ""
	}
	subPath := strings.TrimPrefix(requestPath, basePath)
	return strings.Trim(strings.TrimPrefix(subPath, "/"), "/")
}

func normalizeHookPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultHookIngressPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func resolveAllowedAgentIDs(raw config.FlexibleStringSlice) (map[string]struct{}, bool) {
	if len(raw) == 0 {
		return nil, true
	}
	allowed := make(map[string]struct{}, len(raw))
	for _, id := range raw {
		normalized := strings.TrimSpace(id)
		if normalized == "" {
			continue
		}
		if normalized == "*" {
			return nil, true
		}
		allowed[normalized] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil, true
	}
	return allowed, false
}

func resolveAllowedSessionKeyPrefixes(raw config.FlexibleStringSlice) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, prefix := range raw {
		normalized := strings.ToLower(strings.TrimSpace(prefix))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isSessionKeyAllowedByPrefix(sessionKey string, prefixes []string) bool {
	normalized := strings.ToLower(strings.TrimSpace(sessionKey))
	if normalized == "" {
		return false
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func newHookSessionKey() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hookSessionPrefix + hex.EncodeToString(buf[:])
	}
	return hookSessionPrefix + fmt.Sprintf("%d", time.Now().UnixNano())
}

func resolveHookClientKey(req *http.Request) string {
	if ip := normalizeHookClientIP(req.RemoteAddr); ip != "" {
		return ip
	}
	return "unknown"
}

func normalizeHookClientIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	return strings.TrimSpace(raw)
}

func boolField(raw any, def bool) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "false", "0", "no", "off":
			return false
		case "true", "1", "yes", "on":
			return true
		}
	}
	return def
}

func intField(raw any, def int) int {
	switch v := raw.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case float64:
		if v > 0 {
			return int(v)
		}
	case json.Number:
		if n, err := v.Int64(); err == nil && n > 0 {
			return int(n)
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func stringField(raw any) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func normalizeHookHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		out[strings.ToLower(key)] = strings.Join(values, ", ")
	}
	return out
}

func boolPtr(v bool) *bool {
	return &v
}

func boolValue(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func hookSecretsEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
