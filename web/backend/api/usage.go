package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/providers"
)

type usageRoleCounts struct {
	User      int `json:"user"`
	Assistant int `json:"assistant"`
	Tool      int `json:"tool"`
	System    int `json:"system"`
	Other     int `json:"other"`
}

type usageTotals struct {
	Sessions        int             `json:"sessions"`
	Messages        int             `json:"messages"`
	UserMessages    int             `json:"user_messages"`
	AssistantMsgs   int             `json:"assistant_messages"`
	ToolCalls       int             `json:"tool_calls"`
	EstimatedChars  int             `json:"estimated_chars"`
	EstimatedTokens int             `json:"estimated_tokens"`
	RoleCounts      usageRoleCounts `json:"role_counts"`
}

type usageDailyBucket struct {
	Date            string `json:"date"`
	Sessions        int    `json:"sessions"`
	Messages        int    `json:"messages"`
	ToolCalls       int    `json:"tool_calls"`
	EstimatedChars  int    `json:"estimated_chars"`
	EstimatedTokens int    `json:"estimated_tokens"`
}

type usageSessionItem struct {
	ID              string          `json:"id"`
	Key             string          `json:"key"`
	Title           string          `json:"title"`
	Preview         string          `json:"preview"`
	Created         string          `json:"created"`
	Updated         string          `json:"updated"`
	MessageCount    int             `json:"message_count"`
	ToolCalls       int             `json:"tool_calls"`
	EstimatedChars  int             `json:"estimated_chars"`
	EstimatedTokens int             `json:"estimated_tokens"`
	Roles           usageRoleCounts `json:"roles"`
}

type usageLogItem struct {
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolCalls int    `json:"tool_calls"`
	Updated   string `json:"updated"`
}

func (h *Handler) registerUsageRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/usage", h.handleUsage)
}

func (h *Handler) handleUsage(w http.ResponseWriter, r *http.Request) {
	sessions := h.listAllSessions()
	startDate, _ := parseDateParam(r.URL.Query().Get("start"))
	endDate, _ := parseDateParam(r.URL.Query().Get("end"))
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

	totals := usageTotals{}
	daily := map[string]*usageDailyBucket{}
	sessionItems := make([]usageSessionItem, 0, len(sessions))
	logs := []usageLogItem{}

	for _, sess := range sessions {
		if !dateInRange(sess.Updated, startDate, endDate) {
			continue
		}
		sessionID, ok := extractJameSessionID(sess.Key)
		if !ok {
			continue
		}
		item := buildUsageSessionItem(sessionID, sess)
		if query != "" && !usageSessionMatches(item, sess.Messages, query) {
			continue
		}

		totals.Sessions++
		totals.Messages += item.MessageCount
		totals.UserMessages += item.Roles.User
		totals.AssistantMsgs += item.Roles.Assistant
		totals.ToolCalls += item.ToolCalls
		totals.EstimatedChars += item.EstimatedChars
		totals.EstimatedTokens += item.EstimatedTokens
		totals.RoleCounts.User += item.Roles.User
		totals.RoleCounts.Assistant += item.Roles.Assistant
		totals.RoleCounts.Tool += item.Roles.Tool
		totals.RoleCounts.System += item.Roles.System
		totals.RoleCounts.Other += item.Roles.Other

		day := sess.Updated.Local().Format("2006-01-02")
		bucket := daily[day]
		if bucket == nil {
			bucket = &usageDailyBucket{Date: day}
			daily[day] = bucket
		}
		bucket.Sessions++
		bucket.Messages += item.MessageCount
		bucket.ToolCalls += item.ToolCalls
		bucket.EstimatedChars += item.EstimatedChars
		bucket.EstimatedTokens += item.EstimatedTokens

		sessionItems = append(sessionItems, item)
		logs = append(logs, buildUsageLogs(sessionID, sess)...)
	}

	dailyItems := make([]usageDailyBucket, 0, len(daily))
	for _, bucket := range daily {
		dailyItems = append(dailyItems, *bucket)
	}
	sort.Slice(dailyItems, func(i, j int) bool {
		return dailyItems[i].Date < dailyItems[j].Date
	})
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Updated > logs[j].Updated
	})
	if len(logs) > 200 {
		logs = logs[:200]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"totals":   totals,
		"daily":    dailyItems,
		"sessions": sessionItems,
		"logs":     logs,
	})
}

func parseDateParam(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", value, time.Local)
	return t, err == nil
}

func dateInRange(value time.Time, start time.Time, end time.Time) bool {
	if value.IsZero() {
		return true
	}
	local := value.Local()
	if !start.IsZero() && local.Before(start) {
		return false
	}
	if !end.IsZero() && local.After(end.Add(24*time.Hour-time.Nanosecond)) {
		return false
	}
	return true
}

func buildUsageSessionItem(sessionID string, sess sessionFile) usageSessionItem {
	listItem := buildSessionListItem(sessionID, sess)
	item := usageSessionItem{
		ID:      sessionID,
		Key:     sess.Key,
		Title:   listItem.Title,
		Preview: listItem.Preview,
		Created: listItem.Created,
		Updated: listItem.Updated,
	}
	for _, msg := range sess.Messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" && len(msg.ToolCalls) == 0 {
			continue
		}
		item.MessageCount++
		item.EstimatedChars += len([]rune(content))
		item.ToolCalls += len(msg.ToolCalls)
		switch msg.Role {
		case "user":
			item.Roles.User++
		case "assistant":
			item.Roles.Assistant++
		case "tool":
			item.Roles.Tool++
		case "system":
			item.Roles.System++
		default:
			item.Roles.Other++
		}
	}
	item.EstimatedTokens = estimateTokens(item.EstimatedChars)
	return item
}

func buildUsageLogs(sessionID string, sess sessionFile) []usageLogItem {
	logs := []usageLogItem{}
	for _, msg := range sess.Messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" && len(msg.ToolCalls) == 0 {
			continue
		}
		logs = append(logs, usageLogItem{
			SessionID: sessionID,
			Role:      msg.Role,
			Content:   truncateRunes(content, 220),
			ToolCalls: len(msg.ToolCalls),
			Updated:   sess.Updated.Format(time.RFC3339),
		})
	}
	return logs
}

func usageSessionMatches(item usageSessionItem, messages []providers.Message, query string) bool {
	if strings.Contains(strings.ToLower(item.Title), query) ||
		strings.Contains(strings.ToLower(item.Preview), query) ||
		strings.Contains(strings.ToLower(item.ID), query) {
		return true
	}
	for _, msg := range messages {
		if strings.Contains(strings.ToLower(msg.Role), query) ||
			strings.Contains(strings.ToLower(msg.Content), query) {
			return true
		}
		for _, call := range msg.ToolCalls {
			if call.Function != nil && strings.Contains(strings.ToLower(call.Function.Name), query) {
				return true
			}
		}
	}
	return false
}

func estimateTokens(chars int) int {
	if chars <= 0 {
		return 0
	}
	return (chars + 3) / 4
}
