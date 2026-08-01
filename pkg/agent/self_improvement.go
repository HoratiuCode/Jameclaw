package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/jameclaw/pkg/fileutil"
	"github.com/sipeed/jameclaw/pkg/utils"
)

const (
	selfImprovementVersion        = 1
	maxSelfImprovementReflections = 200
)

var ErrProtectedImprovement = errors.New("protected security or permission behavior cannot become an autonomous skill")

var selfImprovementLocks sync.Map

type LearningCandidate struct {
	ID               string   `json:"id"`
	Kind             string   `json:"kind"`
	Title            string   `json:"title"`
	Lesson           string   `json:"lesson"`
	Evidence         string   `json:"evidence"`
	SourceSession    string   `json:"source_session,omitempty"`
	Scope            string   `json:"scope"`
	Confidence       float64  `json:"confidence"`
	Status           string   `json:"status"`
	Occurrences      int      `json:"occurrences"`
	RequiresApproval bool     `json:"requires_approval"`
	Tools            []string `json:"tools,omitempty"`
	SkillPath        string   `json:"skill_path,omitempty"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type TurnReflection struct {
	ID            string   `json:"id"`
	Session       string   `json:"session"`
	Objective     string   `json:"objective"`
	Outcome       string   `json:"outcome"`
	ResultSummary string   `json:"result_summary"`
	Tools         []string `json:"tools,omitempty"`
	ToolFailures  []string `json:"tool_failures,omitempty"`
	Corrections   int      `json:"corrections"`
	Fingerprint   string   `json:"fingerprint"`
	CreatedAt     string   `json:"created_at"`
}

type TurnLearningInput struct {
	Session      string
	UserMessage  string
	FinalContent string
	Tools        []string
	ToolFailures []string
}

type SelfImprovementMetrics struct {
	Reflections          int     `json:"reflections"`
	CompletedTasks       int     `json:"completed_tasks"`
	TasksWithToolErrors  int     `json:"tasks_with_tool_errors"`
	CorrectionSignals    int     `json:"correction_signals"`
	PendingCandidates    int     `json:"pending_candidates"`
	PromotedCandidates   int     `json:"promoted_candidates"`
	RejectedCandidates   int     `json:"rejected_candidates"`
	StaleCandidates      int     `json:"stale_candidates"`
	SkillsCreated        int     `json:"skills_created"`
	CompletionRate       float64 `json:"completion_rate"`
	CorrectionRate       float64 `json:"correction_rate"`
	RepeatedFailureCount int     `json:"repeated_failure_count"`
}

type SelfImprovementSnapshot struct {
	Candidates  []LearningCandidate    `json:"candidates"`
	Reflections []TurnReflection       `json:"reflections"`
	Metrics     SelfImprovementMetrics `json:"metrics"`
}

type selfImprovementData struct {
	Version     int                 `json:"version"`
	Candidates  []LearningCandidate `json:"candidates"`
	Reflections []TurnReflection    `json:"reflections"`
}

type SelfImprovementStore struct {
	workspace  string
	path       string
	memoryPath string
}

func NewSelfImprovementStore(workspace string) *SelfImprovementStore {
	workspace = filepath.Clean(strings.TrimSpace(workspace))
	return &SelfImprovementStore{
		workspace:  workspace,
		path:       filepath.Join(workspace, "memory", "self-improvement.json"),
		memoryPath: filepath.Join(workspace, "memory", "MEMORY.md"),
	}
}

func (s *SelfImprovementStore) RecordTurn(input TurnLearningInput) error {
	if s == nil || s.workspace == "" || strings.TrimSpace(input.UserMessage) == "" {
		return nil
	}
	return s.withData(func(data *selfImprovementData) error {
		now := time.Now().UTC().Format(time.RFC3339)
		correction := correctionSignal(input.UserMessage)
		reflection := TurnReflection{
			ID:            improvementID("reflection", input.Session+now+input.UserMessage),
			Session:       input.Session,
			Objective:     utils.Truncate(strings.TrimSpace(input.UserMessage), 800),
			Outcome:       turnOutcome(input.FinalContent, input.ToolFailures),
			ResultSummary: utils.Truncate(strings.TrimSpace(input.FinalContent), 1000),
			Tools:         uniqueSorted(input.Tools),
			ToolFailures:  uniqueSorted(input.ToolFailures),
			Fingerprint:   workflowFingerprint(input.UserMessage, input.Tools),
			CreatedAt:     now,
		}
		if correction != "" {
			reflection.Corrections = 1
		}
		data.Reflections = append(data.Reflections, reflection)
		if len(data.Reflections) > maxSelfImprovementReflections {
			data.Reflections = data.Reflections[len(data.Reflections)-maxSelfImprovementReflections:]
		}

		if lesson, explicit := explicitMemoryLesson(input.UserMessage); lesson != "" {
			candidate := LearningCandidate{
				Kind:             "preference",
				Title:            "User preference",
				Lesson:           lesson,
				Evidence:         utils.Truncate(input.UserMessage, 600),
				SourceSession:    input.Session,
				Scope:            "global",
				Confidence:       0.82,
				Status:           "pending",
				RequiresApproval: true,
			}
			index := mergeCandidate(data, candidate, now)
			if explicit && !protectedImprovement(candidate.Lesson) {
				data.Candidates[index].Status = "promoted"
				data.Candidates[index].RequiresApproval = false
				data.Candidates[index].Confidence = 1
				data.Candidates[index].UpdatedAt = now
				if err := s.appendPromotedMemory(data.Candidates[index]); err != nil {
					return err
				}
			}
		}

		if correction != "" {
			mergeCandidate(data, LearningCandidate{
				Kind:             "correction",
				Title:            "Correction from the user",
				Lesson:           correction,
				Evidence:         utils.Truncate(input.UserMessage, 600),
				SourceSession:    input.Session,
				Scope:            "similar tasks",
				Confidence:       0.78,
				Status:           "pending",
				RequiresApproval: true,
			}, now)
		}

		for _, failure := range uniqueSorted(input.ToolFailures) {
			mergeCandidate(data, LearningCandidate{
				Kind:             "tool_lesson",
				Title:            "Repeated tool failure to avoid",
				Lesson:           "Before retrying a similar tool action, account for this failure: " + utils.Truncate(failure, 420),
				Evidence:         utils.Truncate(failure, 500),
				SourceSession:    input.Session,
				Scope:            "tool workflow",
				Confidence:       0.55,
				Status:           "pending",
				RequiresApproval: true,
				Tools:            uniqueSorted(input.Tools),
			}, now)
		}

		if reflection.Outcome == "completed" && len(reflection.Tools) > 0 {
			matches := 0
			for _, previous := range data.Reflections {
				if previous.Fingerprint == reflection.Fingerprint && previous.Outcome == "completed" {
					matches++
				}
			}
			if matches >= 2 {
				mergeCandidate(data, LearningCandidate{
					Kind:             "workflow",
					Title:            "Reusable workflow detected",
					Lesson:           "For requests similar to “" + utils.Truncate(strings.TrimSpace(input.UserMessage), 260) + "”, reuse the verified tool sequence: " + strings.Join(reflection.Tools, " → ") + ". Verify the final result before reporting completion.",
					Evidence:         fmt.Sprintf("Completed successfully %d times with %s.", matches, strings.Join(reflection.Tools, ", ")),
					SourceSession:    input.Session,
					Scope:            "repeated workflow",
					Confidence:       min(0.95, 0.55+float64(matches)*0.1),
					Status:           "pending",
					RequiresApproval: true,
					Tools:            reflection.Tools,
				}, now)
			}
		}
		return nil
	})
}

func (s *SelfImprovementStore) Snapshot() (SelfImprovementSnapshot, error) {
	var snapshot SelfImprovementSnapshot
	err := s.readData(func(data *selfImprovementData) error {
		snapshot.Candidates = append([]LearningCandidate(nil), data.Candidates...)
		snapshot.Reflections = append([]TurnReflection(nil), data.Reflections...)
		sort.SliceStable(snapshot.Candidates, func(i, j int) bool {
			return snapshot.Candidates[i].UpdatedAt > snapshot.Candidates[j].UpdatedAt
		})
		sort.SliceStable(snapshot.Reflections, func(i, j int) bool {
			return snapshot.Reflections[i].CreatedAt > snapshot.Reflections[j].CreatedAt
		})
		snapshot.Metrics = improvementMetrics(data)
		return nil
	})
	return snapshot, err
}

func (s *SelfImprovementStore) readData(read func(*selfImprovementData) error) error {
	lockValue, _ := selfImprovementLocks.LoadOrStore(s.path, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	data, err := s.loadData()
	if err != nil {
		return err
	}
	return read(&data)
}

func (s *SelfImprovementStore) ApplyCandidateAction(id, action, title, lesson string) (LearningCandidate, error) {
	var updated LearningCandidate
	err := s.withData(func(data *selfImprovementData) error {
		index := -1
		for i := range data.Candidates {
			if data.Candidates[i].ID == id {
				index = i
				break
			}
		}
		if index < 0 {
			return os.ErrNotExist
		}
		candidate := &data.Candidates[index]
		now := time.Now().UTC().Format(time.RFC3339)
		switch action {
		case "edit":
			if clean := strings.TrimSpace(title); clean != "" {
				candidate.Title = utils.Truncate(clean, 160)
			}
			if clean := strings.TrimSpace(lesson); clean != "" {
				candidate.Lesson = utils.Truncate(clean, 1600)
			}
			candidate.Status = "pending"
			candidate.RequiresApproval = true
		case "approve":
			candidate.Status = "promoted"
			candidate.RequiresApproval = false
			if err := s.appendPromotedMemory(*candidate); err != nil {
				return err
			}
		case "reject":
			candidate.Status = "rejected"
			candidate.RequiresApproval = false
		case "create_skill":
			if candidate.Status != "promoted" {
				return fmt.Errorf("approve the improvement before creating a skill")
			}
			if protectedImprovement(candidate.Lesson) {
				return ErrProtectedImprovement
			}
			path, err := s.createSkill(*candidate)
			if err != nil {
				return err
			}
			candidate.SkillPath = path
		default:
			return fmt.Errorf("unsupported candidate action %q", action)
		}
		candidate.UpdatedAt = now
		updated = *candidate
		return nil
	})
	return updated, err
}

func (s *SelfImprovementStore) Maintain(now time.Time) error {
	return s.withData(func(data *selfImprovementData) error {
		cutoff := now.AddDate(0, 0, -90)
		for i := range data.Candidates {
			candidate := &data.Candidates[i]
			if candidate.Status != "pending" {
				continue
			}
			updatedAt, err := time.Parse(time.RFC3339, candidate.UpdatedAt)
			if err == nil && updatedAt.Before(cutoff) {
				candidate.Status = "stale"
				candidate.RequiresApproval = true
				candidate.UpdatedAt = now.UTC().Format(time.RFC3339)
			}
		}
		return nil
	})
}

func (s *SelfImprovementStore) withData(update func(*selfImprovementData) error) error {
	lockValue, _ := selfImprovementLocks.LoadOrStore(s.path, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	data, err := s.loadData()
	if err != nil {
		return err
	}
	if err := update(&data); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(s.path, raw, 0o600)
}

func (s *SelfImprovementStore) loadData() (selfImprovementData, error) {
	data := selfImprovementData{Version: selfImprovementVersion, Candidates: []LearningCandidate{}, Reflections: []TurnReflection{}}
	raw, err := os.ReadFile(s.path)
	if err == nil {
		if err := json.Unmarshal(raw, &data); err != nil {
			return selfImprovementData{}, err
		}
	} else if !os.IsNotExist(err) {
		return selfImprovementData{}, err
	}
	if data.Version == 0 {
		data.Version = selfImprovementVersion
	}
	if data.Candidates == nil {
		data.Candidates = []LearningCandidate{}
	}
	if data.Reflections == nil {
		data.Reflections = []TurnReflection{}
	}
	return data, nil
}

func (s *SelfImprovementStore) appendPromotedMemory(candidate LearningCandidate) error {
	block := fmt.Sprintf("## Learned Improvement — %s\n\n- %s\n- Source: %s\n", candidate.Title, candidate.Lesson, firstNonEmpty(candidate.SourceSession, "user-approved learning"))
	existing, err := os.ReadFile(s.memoryPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(existing), candidate.Lesson) {
		return nil
	}
	content := strings.TrimSpace(string(existing))
	if content != "" {
		content += "\n\n"
	}
	content += block
	return fileutil.WriteFileAtomic(s.memoryPath, []byte(content), 0o600)
}

func (s *SelfImprovementStore) createSkill(candidate LearningCandidate) (string, error) {
	slug := skillSlug(candidate.Title)
	if slug == "" {
		slug = "learned-workflow-" + candidate.ID[:8]
	}
	dir := filepath.Join(s.workspace, "skills", slug)
	path := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	description := utils.Truncate(strings.ReplaceAll(candidate.Lesson, "\n", " "), 360)
	descriptionJSON, _ := json.Marshal(description)
	content := fmt.Sprintf(`---
name: %s
description: %s
---

# %s

Use this learned workflow only for tasks that match its evidence and scope.

## Learned procedure

%s

## Tools observed

%s

## Verification

Verify the concrete output before reporting completion. If the task differs materially, stop using this workflow and re-plan.
`, slug, string(descriptionJSON), candidate.Title, candidate.Lesson, markdownToolList(candidate.Tools))
	if err := fileutil.WriteFileAtomic(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func mergeCandidate(data *selfImprovementData, candidate LearningCandidate, now string) int {
	fingerprint := improvementFingerprint(candidate.Kind, candidate.Lesson)
	for i := range data.Candidates {
		if improvementFingerprint(data.Candidates[i].Kind, data.Candidates[i].Lesson) != fingerprint {
			continue
		}
		data.Candidates[i].Occurrences++
		data.Candidates[i].Confidence = min(0.98, max(data.Candidates[i].Confidence, candidate.Confidence)+0.05)
		data.Candidates[i].Evidence = candidate.Evidence
		data.Candidates[i].SourceSession = candidate.SourceSession
		data.Candidates[i].UpdatedAt = now
		return i
	}
	candidate.ID = improvementID(candidate.Kind, fingerprint)
	candidate.Occurrences = 1
	candidate.CreatedAt = now
	candidate.UpdatedAt = now
	data.Candidates = append(data.Candidates, candidate)
	return len(data.Candidates) - 1
}

func improvementMetrics(data *selfImprovementData) SelfImprovementMetrics {
	metrics := SelfImprovementMetrics{Reflections: len(data.Reflections)}
	failureCounts := make(map[string]int)
	for _, reflection := range data.Reflections {
		if reflection.Outcome == "completed" {
			metrics.CompletedTasks++
		}
		if len(reflection.ToolFailures) > 0 {
			metrics.TasksWithToolErrors++
		}
		metrics.CorrectionSignals += reflection.Corrections
		for _, failure := range reflection.ToolFailures {
			failureCounts[normalizeImprovementText(failure)]++
		}
	}
	for _, count := range failureCounts {
		if count > 1 {
			metrics.RepeatedFailureCount++
		}
	}
	for _, candidate := range data.Candidates {
		switch candidate.Status {
		case "pending":
			metrics.PendingCandidates++
		case "promoted":
			metrics.PromotedCandidates++
		case "rejected":
			metrics.RejectedCandidates++
		case "stale":
			metrics.StaleCandidates++
		}
		if candidate.SkillPath != "" {
			metrics.SkillsCreated++
		}
	}
	if metrics.Reflections > 0 {
		metrics.CompletionRate = float64(metrics.CompletedTasks) / float64(metrics.Reflections)
		metrics.CorrectionRate = float64(metrics.CorrectionSignals) / float64(metrics.Reflections)
	}
	return metrics
}

func explicitMemoryLesson(message string) (string, bool) {
	clean := strings.TrimSpace(message)
	lower := strings.ToLower(clean)
	for _, prefix := range []string{"remember that ", "please remember that ", "remember this: ", "please remember: "} {
		if index := strings.Index(lower, prefix); index >= 0 {
			lesson := strings.TrimSpace(clean[index+len(prefix):])
			return utils.Truncate(lesson, 1600), lesson != ""
		}
	}
	for _, marker := range []string{"i prefer ", "from now on ", "always use ", "never use ", "whenever i "} {
		if strings.Contains(lower, marker) {
			return utils.Truncate(clean, 1600), false
		}
	}
	return "", false
}

func correctionSignal(message string) string {
	clean := strings.TrimSpace(message)
	lower := strings.ToLower(clean)
	for _, marker := range []string{
		"that is wrong", "this is wrong", "not correct", "not what i asked", "not what i want",
		"you didn't", "you did not", "try again", "redo", "fix this", "doesn't work", "does not work",
		"remove that", "instead,", "i said ", "the problem is", "same problem",
	} {
		if strings.Contains(lower, marker) {
			return "For similar requests, apply this user correction: " + utils.Truncate(clean, 1200)
		}
	}
	return ""
}

func turnOutcome(finalContent string, failures []string) string {
	if strings.TrimSpace(finalContent) == "" {
		return "incomplete"
	}
	if len(failures) > 0 {
		return "completed_with_errors"
	}
	return "completed"
}

func workflowFingerprint(message string, tools []string) string {
	return improvementFingerprint("workflow", normalizeImprovementText(message)+"|"+strings.Join(uniqueSorted(tools), ","))
}

func improvementFingerprint(kind, value string) string {
	hash := sha256.Sum256([]byte(kind + "|" + normalizeImprovementText(value)))
	return hex.EncodeToString(hash[:])
}

func improvementID(kind, value string) string {
	fingerprint := improvementFingerprint(kind, value)
	return kind + "-" + fingerprint[:16]
}

var improvementWhitespace = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeImprovementText(value string) string {
	return strings.TrimSpace(improvementWhitespace.ReplaceAllString(strings.ToLower(value), " "))
}

func protectedImprovement(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"password", "api key", "credential", "private key", "disable security", "disable approval",
		"never ask approval", "yolo mode", "bypass permission", "change permission", "security rule",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func skillSlug(value string) string {
	value = strings.TrimSpace(improvementWhitespace.ReplaceAllString(strings.ToLower(value), "-"))
	value = strings.Trim(value, "-")
	if len(value) > 54 {
		value = strings.Trim(value[:54], "-")
	}
	return value
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func markdownToolList(tools []string) string {
	if len(tools) == 0 {
		return "- No fixed tool requirement was learned."
	}
	lines := make([]string, 0, len(tools))
	for _, tool := range uniqueSorted(tools) {
		lines = append(lines, "- `"+tool+"`")
	}
	return strings.Join(lines, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
