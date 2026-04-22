package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/sipeed/jameclaw/pkg/config"
)

const (
	hookMappingActionWake  = "wake"
	hookMappingActionAgent = "agent"
	hookWakeModeNow        = "now"
	hookWakeModeNext       = "next-heartbeat"
)

type hookMappingResolved struct {
	id              string
	matchPath       string
	matchSource     string
	action          string
	wakeMode        string
	name            string
	agentID         string
	sessionKey      string
	messageTemplate string
	textTemplate    string
	deliver         *bool
	channel         string
	to              string
	timeoutSeconds  int
	transform       *hookMappingTransformResolved
}

type hookMappingTransformResolved struct {
	modulePath string
	exportName string
}

type hookTemplateContext struct {
	Payload map[string]any
	Headers map[string]string
	URL     string
	Path    string
}

type hookAction struct {
	kind           string
	text           string
	mode           string
	message        string
	name           string
	agentID        string
	sessionKey     string
	deliver        *bool
	channel        string
	to             string
	timeoutSeconds int
}

type hookTransformOverride struct {
	Kind           *string `json:"kind,omitempty"`
	Text           *string `json:"text,omitempty"`
	Mode           *string `json:"mode,omitempty"`
	Message        *string `json:"message,omitempty"`
	Name           *string `json:"name,omitempty"`
	AgentID        *string `json:"agentId,omitempty"`
	WakeMode       *string `json:"wakeMode,omitempty"`
	SessionKey     *string `json:"sessionKey,omitempty"`
	Deliver        *bool   `json:"deliver,omitempty"`
	Channel        *string `json:"channel,omitempty"`
	To             *string `json:"to,omitempty"`
	TimeoutSeconds *int    `json:"timeoutSeconds,omitempty"`
}

type hookMappingOutcome struct {
	Action  *hookAction
	Matched bool
	Skipped bool
}

func resolveHookMappings(hooks config.HooksConfig, configDir string) ([]hookMappingResolved, error) {
	mappings := make([]config.WebhookMappingConfig, 0, len(hooks.Mappings)+len(hooks.Presets))
	if len(hooks.Mappings) > 0 {
		mappings = append(mappings, hooks.Mappings...)
	}
	for _, preset := range hooks.Presets {
		switch strings.ToLower(strings.TrimSpace(preset)) {
		case "gmail":
			mappings = append(mappings, config.WebhookMappingConfig{
				ID:              "gmail",
				Match:           config.WebhookMappingMatchConfig{Path: "gmail"},
				Action:          "agent",
				WakeMode:        hookWakeModeNow,
				Name:            "Gmail",
				SessionKey:      "hook:gmail:{{path \"messages.0.id\"}}",
				MessageTemplate: "New email from {{path \"messages.0.from\"}}\nSubject: {{path \"messages.0.subject\"}}\n{{path \"messages.0.snippet\"}}\n{{path \"messages.0.body\"}}",
			})
		}
	}
	if len(mappings) == 0 {
		return nil, nil
	}

	baseDir := strings.TrimSpace(configDir)
	if baseDir == "" {
		baseDir = "."
	}
	baseDir = filepath.Clean(baseDir)
	transformsRoot := filepath.Join(baseDir, "hooks", "transforms")
	transformsDir, err := resolveHookOptionalContainedPath(transformsRoot, hooks.TransformsDir, "Hook transformsDir")
	if err != nil {
		return nil, err
	}

	resolved := make([]hookMappingResolved, 0, len(mappings))
	for i, mapping := range mappings {
		entry, err := normalizeHookMapping(mapping, i, transformsDir)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, entry)
	}
	return resolved, nil
}

func applyHookMappings(
	mappings []hookMappingResolved,
	ctx hookTemplateContext,
) (hookMappingOutcome, error) {
	if len(mappings) == 0 {
		return hookMappingOutcome{}, nil
	}

	for _, mapping := range mappings {
		if !mappingMatches(mapping, ctx) {
			continue
		}

		base, err := buildHookAction(mapping, ctx)
		if err != nil {
			return hookMappingOutcome{}, err
		}

		if mapping.transform != nil {
			override, err := loadHookTransform(mapping.transform, ctx)
			if err != nil {
				return hookMappingOutcome{}, err
			}
			if override == nil {
				return hookMappingOutcome{Matched: true, Skipped: true}, nil
			}
			merged, err := mergeHookAction(base, override)
			if err != nil {
				return hookMappingOutcome{}, err
			}
			return hookMappingOutcome{Action: &merged, Matched: true}, nil
		}

		if err := validateHookAction(base); err != nil {
			return hookMappingOutcome{}, err
		}
		return hookMappingOutcome{Action: &base, Matched: true}, nil
	}

	return hookMappingOutcome{}, nil
}

func normalizeHookMapping(
	mapping config.WebhookMappingConfig,
	index int,
	transformsDir string,
) (hookMappingResolved, error) {
	id := strings.TrimSpace(mapping.ID)
	if id == "" {
		id = fmt.Sprintf("mapping-%d", index+1)
	}

	action := strings.ToLower(strings.TrimSpace(mapping.Action))
	if action == "" {
		action = hookMappingActionAgent
	}
	if action != hookMappingActionWake && action != hookMappingActionAgent {
		return hookMappingResolved{}, fmt.Errorf("hook mapping %q has invalid action %q", id, mapping.Action)
	}

	wakeMode := normalizeHookWakeMode(mapping.WakeMode)
	transform, err := normalizeHookTransform(mapping.Transform, transformsDir)
	if err != nil {
		return hookMappingResolved{}, fmt.Errorf("hook mapping %q: %w", id, err)
	}

	return hookMappingResolved{
		id:              id,
		matchPath:       normalizeHookMatchPath(mapping.Match.Path),
		matchSource:     strings.TrimSpace(mapping.Match.Source),
		action:          action,
		wakeMode:        wakeMode,
		name:            strings.TrimSpace(mapping.Name),
		agentID:         strings.TrimSpace(mapping.AgentID),
		sessionKey:      mapping.SessionKey,
		messageTemplate: mapping.MessageTemplate,
		textTemplate:    mapping.TextTemplate,
		deliver:         mapping.Deliver,
		channel:         strings.TrimSpace(mapping.Channel),
		to:              mapping.To,
		timeoutSeconds:  mapping.TimeoutSeconds,
		transform:       transform,
	}, nil
}

func normalizeHookTransform(
	transform *config.WebhookMappingTransformConfig,
	transformsDir string,
) (*hookMappingTransformResolved, error) {
	if transform == nil {
		return nil, nil
	}
	modulePath, err := resolveHookContainedPath(transformsDir, transform.Module, "Hook transform")
	if err != nil {
		return nil, err
	}
	return &hookMappingTransformResolved{
		modulePath: modulePath,
		exportName: strings.TrimSpace(transform.Export),
	}, nil
}

func mappingMatches(mapping hookMappingResolved, ctx hookTemplateContext) bool {
	if mapping.matchPath != "" && mapping.matchPath != normalizeHookMatchPath(ctx.Path) {
		return false
	}
	if mapping.matchSource != "" {
		source := stringFromHookPayload(ctx.Payload["source"])
		if source == "" || source != mapping.matchSource {
			return false
		}
	}
	return true
}

func buildHookAction(mapping hookMappingResolved, ctx hookTemplateContext) (hookAction, error) {
	if mapping.action == hookMappingActionWake {
		text, err := renderHookTemplate(mapping.textTemplate, ctx)
		if err != nil {
			return hookAction{}, err
		}
		return hookAction{
			kind:           hookMappingActionWake,
			text:           text,
			mode:           mapping.wakeMode,
			timeoutSeconds: mapping.timeoutSeconds,
		}, nil
	}

	message, err := renderHookTemplate(mapping.messageTemplate, ctx)
	if err != nil {
		return hookAction{}, err
	}
	name, err := renderOptionalHookTemplate(mapping.name, ctx)
	if err != nil {
		return hookAction{}, err
	}
	sessionKey, err := renderOptionalHookTemplate(mapping.sessionKey, ctx)
	if err != nil {
		return hookAction{}, err
	}
	to, err := renderOptionalHookTemplate(mapping.to, ctx)
	if err != nil {
		return hookAction{}, err
	}

	return hookAction{
		kind:           hookMappingActionAgent,
		message:        message,
		name:           name,
		agentID:        strings.TrimSpace(mapping.agentID),
		sessionKey:     sessionKey,
		deliver:        mapping.deliver,
		channel:        strings.TrimSpace(mapping.channel),
		to:             to,
		timeoutSeconds: mapping.timeoutSeconds,
	}, nil
}

func mergeHookAction(base hookAction, override *hookTransformOverride) (hookAction, error) {
	if override == nil {
		return base, validateHookAction(base)
	}

	kind := strings.ToLower(strings.TrimSpace(base.kind))
	if override.Kind != nil && strings.TrimSpace(*override.Kind) != "" {
		kind = strings.ToLower(strings.TrimSpace(*override.Kind))
	}
	if kind == "" {
		kind = hookMappingActionAgent
	}
	if kind != hookMappingActionWake && kind != hookMappingActionAgent {
		return hookAction{}, fmt.Errorf("invalid transform kind %q", kind)
	}

	if kind == hookMappingActionWake {
		text := base.text
		if override.Text != nil {
			text = strings.TrimSpace(*override.Text)
		}
		if text == "" && override.Message != nil {
			text = strings.TrimSpace(*override.Message)
		}
		mode := normalizeHookWakeMode(base.mode)
		if override.Mode != nil && strings.TrimSpace(*override.Mode) != "" {
			mode = normalizeHookWakeMode(*override.Mode)
		}
		if override.WakeMode != nil && strings.TrimSpace(*override.WakeMode) != "" {
			mode = normalizeHookWakeMode(*override.WakeMode)
		}
		timeoutSeconds := base.timeoutSeconds
		if override.TimeoutSeconds != nil && *override.TimeoutSeconds > 0 {
			timeoutSeconds = *override.TimeoutSeconds
		}
		result := hookAction{
			kind:           hookMappingActionWake,
			text:           text,
			mode:           mode,
			timeoutSeconds: timeoutSeconds,
		}
		return result, validateHookAction(result)
	}

	message := base.message
	if override.Message != nil {
		message = strings.TrimSpace(*override.Message)
	}
	if message == "" && override.Text != nil {
		message = strings.TrimSpace(*override.Text)
	}
	name := base.name
	if override.Name != nil {
		name = strings.TrimSpace(*override.Name)
	}
	agentID := base.agentID
	if override.AgentID != nil {
		agentID = strings.TrimSpace(*override.AgentID)
	}
	sessionKey := base.sessionKey
	if override.SessionKey != nil {
		sessionKey = strings.TrimSpace(*override.SessionKey)
	}
	deliver := base.deliver
	if override.Deliver != nil {
		deliver = override.Deliver
	}
	channel := base.channel
	if override.Channel != nil {
		channel = strings.TrimSpace(*override.Channel)
	}
	to := base.to
	if override.To != nil {
		to = strings.TrimSpace(*override.To)
	}
	timeoutSeconds := base.timeoutSeconds
	if override.TimeoutSeconds != nil && *override.TimeoutSeconds > 0 {
		timeoutSeconds = *override.TimeoutSeconds
	}

	result := hookAction{
		kind:           hookMappingActionAgent,
		message:        message,
		name:           name,
		agentID:        agentID,
		sessionKey:     sessionKey,
		deliver:        deliver,
		channel:        channel,
		to:             to,
		timeoutSeconds: timeoutSeconds,
	}
	return result, validateHookAction(result)
}

func validateHookAction(action hookAction) error {
	switch action.kind {
	case hookMappingActionWake:
		if strings.TrimSpace(action.text) == "" {
			return fmt.Errorf("hook mapping requires text")
		}
	case hookMappingActionAgent:
		if strings.TrimSpace(action.message) == "" {
			return fmt.Errorf("hook mapping requires message")
		}
	default:
		return fmt.Errorf("hook mapping has unsupported action %q", action.kind)
	}
	return nil
}

func loadHookTransform(
	transform *hookMappingTransformResolved,
	ctx hookTemplateContext,
) (*hookTransformOverride, error) {
	if transform == nil {
		return nil, nil
	}

	content, err := os.ReadFile(transform.modulePath)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(filepath.Base(transform.modulePath)).
		Funcs(hookTemplateFuncs(&ctx)).
		Option("missingkey=zero").
		Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse hook transform %s: %w", transform.modulePath, err)
	}

	var rendered bytes.Buffer
	data := map[string]any{
		"Payload": ctx.Payload,
		"Headers": ctx.Headers,
		"URL":     ctx.URL,
		"Path":    ctx.Path,
	}
	if transform.exportName != "" {
		err = tmpl.ExecuteTemplate(&rendered, transform.exportName, data)
	} else {
		err = tmpl.Execute(&rendered, data)
	}
	if err != nil {
		return nil, fmt.Errorf("hook transform %s: %w", transform.modulePath, err)
	}

	output := strings.TrimSpace(rendered.String())
	if output == "" {
		return nil, nil
	}

	var override hookTransformOverride
	if err := json.Unmarshal([]byte(output), &override); err != nil {
		return nil, fmt.Errorf("hook transform %s returned invalid JSON: %w", transform.modulePath, err)
	}
	return &override, nil
}

func renderHookTemplate(raw string, ctx hookTemplateContext) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	tmpl, err := template.New("hook-template").
		Funcs(hookTemplateFuncs(&ctx)).
		Option("missingkey=zero").
		Parse(raw)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, map[string]any{
		"Payload": ctx.Payload,
		"Headers": ctx.Headers,
		"URL":     ctx.URL,
		"Path":    ctx.Path,
	}); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func renderOptionalHookTemplate(raw string, ctx hookTemplateContext) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return renderHookTemplate(raw, ctx)
}

func hookTemplateFuncs(ctx *hookTemplateContext) template.FuncMap {
	return template.FuncMap{
		"path": func(expr string) any {
			if ctx == nil {
				return ""
			}
			value, ok := lookupHookValue(ctx.Payload, expr)
			if !ok {
				return ""
			}
			return value
		},
		"header": func(name string) string {
			if ctx == nil {
				return ""
			}
			return ctx.Headers[strings.ToLower(strings.TrimSpace(name))]
		},
		"json": func(v any) string {
			data, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(data)
		},
	}
}

func lookupHookValue(root any, expr string) (any, bool) {
	tokens := splitHookPath(expr)
	if len(tokens) == 0 {
		return nil, false
	}

	current := root
	for _, token := range tokens {
		next, ok := advanceHookValue(current, token)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func splitHookPath(expr string) []string {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil
	}

	var tokens []string
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}

	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '.':
			flush()
		case '[':
			flush()
			end := strings.IndexByte(expr[i+1:], ']')
			if end < 0 {
				continue
			}
			token := strings.TrimSpace(expr[i+1 : i+1+end])
			token = strings.Trim(token, `"'`)
			if token != "" {
				tokens = append(tokens, token)
			}
			i += end + 1
		default:
			current.WriteByte(expr[i])
		}
	}
	flush()
	return tokens
}

func advanceHookValue(current any, token string) (any, bool) {
	if current == nil {
		return nil, false
	}

	switch v := current.(type) {
	case map[string]any:
		next, ok := v[token]
		return next, ok
	case map[string]string:
		next, ok := v[token]
		return next, ok
	case []any:
		idx, err := strconv.Atoi(token)
		if err != nil || idx < 0 || idx >= len(v) {
			return nil, false
		}
		return v[idx], true
	case []string:
		idx, err := strconv.Atoi(token)
		if err != nil || idx < 0 || idx >= len(v) {
			return nil, false
		}
		return v[idx], true
	case json.Number:
		if token == "" {
			return v, true
		}
		return nil, false
	}

	return reflectHookValue(current, token)
}

func reflectHookValue(current any, token string) (any, bool) {
	rv := reflect.ValueOf(current)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil, false
	}

	switch rv.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(token)
		value := rv.MapIndex(key)
		if !value.IsValid() {
			return nil, false
		}
		return value.Interface(), true
	case reflect.Slice, reflect.Array:
		idx, err := strconv.Atoi(token)
		if err != nil || idx < 0 || idx >= rv.Len() {
			return nil, false
		}
		return rv.Index(idx).Interface(), true
	case reflect.Struct:
		field := rv.FieldByNameFunc(func(name string) bool {
			return strings.EqualFold(name, token)
		})
		if !field.IsValid() {
			return nil, false
		}
		return field.Interface(), true
	default:
		return nil, false
	}
}

func normalizeHookMatchPath(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "/")
	return raw
}

func normalizeHookWakeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case hookWakeModeNext:
		return hookWakeModeNext
	default:
		return hookWakeModeNow
	}
}

func normalizeHookContainedPath(baseDir, target, label string) (string, error) {
	baseDir = filepath.Clean(baseDir)
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("%s module path is required", label)
	}
	target = expandHookHomePath(target)

	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(baseDir, resolved)
	}
	resolved = filepath.Clean(resolved)
	if escapesHookBase(baseDir, resolved) {
		return "", fmt.Errorf("%s module path must be within %s: %s", label, baseDir, target)
	}

	baseReal, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		baseReal = baseDir
	}
	ancestor := resolved
	for {
		if _, statErr := os.Stat(ancestor); statErr == nil {
			break
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			ancestor = ""
			break
		}
		ancestor = parent
	}
	if ancestor != "" {
		ancestorReal, err := filepath.EvalSymlinks(ancestor)
		if err == nil && escapesHookBase(baseReal, ancestorReal) {
			return "", fmt.Errorf("%s module path must be within %s: %s", label, baseDir, target)
		}
	}
	return resolved, nil
}

func expandHookHomePath(target string) string {
	if target == "~" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return home
		}
		return target
	}
	if strings.HasPrefix(target, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, target[2:])
		}
	}
	return target
}

func resolveHookContainedPath(baseDir, target, label string) (string, error) {
	return normalizeHookContainedPath(baseDir, target, label)
}

func resolveHookOptionalContainedPath(baseDir, target, label string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return filepath.Clean(baseDir), nil
	}
	return normalizeHookContainedPath(baseDir, target, label)
}

func escapesHookBase(baseDir, candidate string) bool {
	relative, err := filepath.Rel(baseDir, candidate)
	if err != nil {
		return true
	}
	return relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative)
}

func stringFromHookPayload(raw any) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}
