package cron

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/adhocore/gronx"

	"github.com/sipeed/jameclaw/pkg/fileutil"
)

type CronSchedule struct {
	Kind    string `json:"kind"`
	AtMS    *int64 `json:"atMs,omitempty"`
	EveryMS *int64 `json:"everyMs,omitempty"`
	Expr    string `json:"expr,omitempty"`
	TZ      string `json:"tz,omitempty"`
}

type CronPayload struct {
	Kind             string `json:"kind"`
	Message          string `json:"message"`
	Command          string `json:"command,omitempty"`
	Deliver          bool   `json:"deliver"`
	DeliveryApproved bool   `json:"delivery_approved,omitempty"`
	Channel          string `json:"channel,omitempty"`
	To               string `json:"to,omitempty"`
}

type CronJobState struct {
	NextRunAtMS         *int64 `json:"nextRunAtMs,omitempty"`
	LastRunAtMS         *int64 `json:"lastRunAtMs,omitempty"`
	RunningAtMS         *int64 `json:"runningAtMs,omitempty"`
	RunClaimExpiresAtMS *int64 `json:"runClaimExpiresAtMs,omitempty"`
	LastStatus          string `json:"lastStatus,omitempty"`
	LastError           string `json:"lastError,omitempty"`
	LastOutputPath      string `json:"lastOutputPath,omitempty"`
	LastDurationMS      int64  `json:"lastDurationMs,omitempty"`
}

type CronJob struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Enabled        bool         `json:"enabled"`
	Schedule       CronSchedule `json:"schedule"`
	Payload        CronPayload  `json:"payload"`
	State          CronJobState `json:"state"`
	CreatedAtMS    int64        `json:"createdAtMs"`
	UpdatedAtMS    int64        `json:"updatedAtMs"`
	DeleteAfterRun bool         `json:"deleteAfterRun"`
}

type CronStore struct {
	Version int       `json:"version"`
	Jobs    []CronJob `json:"jobs"`
}

type JobHandler func(job *CronJob) (string, error)

const (
	storeVersion              = 2
	defaultParallelJobs       = 4
	runClaimTTL               = 30 * time.Minute
	tickerHeartbeatFilename   = "ticker_heartbeat"
	tickerLastSuccessFilename = "ticker_last_success"
)

type CronService struct {
	storePath         string
	cronDir           string
	outputDir         string
	lockPath          string
	heartbeatPath     string
	lastSuccessPath   string
	store             *CronStore
	onJob             JobHandler
	mu                sync.RWMutex
	running           bool
	stopChan          chan struct{}
	wakeChan          chan struct{}
	gronx             *gronx.Gronx
	parallelSem       chan struct{}
	parallelWaitGroup sync.WaitGroup
}

func NewCronService(storePath string, onJob JobHandler) *CronService {
	cronDir := filepath.Dir(storePath)
	cs := &CronService{
		storePath:       storePath,
		cronDir:         cronDir,
		outputDir:       filepath.Join(cronDir, "output"),
		lockPath:        filepath.Join(cronDir, ".jobs.lock"),
		heartbeatPath:   filepath.Join(cronDir, tickerHeartbeatFilename),
		lastSuccessPath: filepath.Join(cronDir, tickerLastSuccessFilename),
		onJob:           onJob,
		gronx:           gronx.New(),
		wakeChan:        make(chan struct{}),
		parallelSem:     make(chan struct{}, defaultParallelJobs),
	}
	// Initialize and load store on creation
	cs.loadStore()
	return cs
}

func (cs *CronService) Start() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.running {
		return nil
	}

	if err := cs.loadStore(); err != nil {
		return fmt.Errorf("failed to load store: %w", err)
	}

	cs.recoverStaleRunningJobs(time.Now().UnixMilli())
	cs.recomputeNextRuns()
	if err := cs.saveStoreUnsafe(); err != nil {
		return fmt.Errorf("failed to save store: %w", err)
	}

	cs.stopChan = make(chan struct{})
	if cs.wakeChan == nil {
		cs.wakeChan = make(chan struct{})
	}
	cs.running = true
	go cs.runLoop(cs.stopChan)

	return nil
}

func (cs *CronService) Stop() {
	cs.mu.Lock()

	if !cs.running {
		cs.mu.Unlock()
		return
	}

	cs.running = false
	if cs.stopChan != nil {
		close(cs.stopChan)
		cs.stopChan = nil
	}
	cs.mu.Unlock()
	cs.parallelWaitGroup.Wait()
}

func (cs *CronService) runLoop(stopChan chan struct{}) {
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {
		cs.writeHeartbeat(cs.heartbeatPath)
		// every loop, recalculate the next wake time
		cs.mu.RLock()
		nextWake := cs.getNextWakeMS()
		cs.mu.RUnlock()

		var delay time.Duration
		now := time.Now().UnixMilli()

		if nextWake == nil {
			// no jobs, sleep for a long time (or until a new job is added)
			delay = time.Hour
		} else {
			diff := *nextWake - now
			if diff <= 0 {
				delay = 0
			} else {
				delay = time.Duration(diff) * time.Millisecond
			}
		}

		timer.Reset(delay)

		select {
		case <-stopChan:
			return
		case <-cs.wakeChan: // wake on new job or update
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			continue
		case <-timer.C:
			cs.checkJobs()
			cs.writeHeartbeat(cs.lastSuccessPath)
		}
	}
}

func (cs *CronService) checkJobs() {
	cs.mu.Lock()

	if !cs.running {
		cs.mu.Unlock()
		return
	}

	now := time.Now().UnixMilli()
	var dueJobIDs []string
	cs.recoverStaleRunningJobs(now)

	// Collect jobs that are due (we need to copy them to execute outside lock)
	for i := range cs.store.Jobs {
		job := &cs.store.Jobs[i]
		if job.Enabled && job.State.RunningAtMS == nil && job.State.NextRunAtMS != nil && *job.State.NextRunAtMS <= now {
			dueJobIDs = append(dueJobIDs, job.ID)
		}
	}

	// Reset next run for due jobs before unlocking to avoid duplicate execution.
	dueMap := make(map[string]bool, len(dueJobIDs))
	for _, jobID := range dueJobIDs {
		dueMap[jobID] = true
	}
	for i := range cs.store.Jobs {
		if dueMap[cs.store.Jobs[i].ID] {
			cs.store.Jobs[i].State.NextRunAtMS = nil
			cs.store.Jobs[i].State.RunningAtMS = cronInt64Ptr(now)
			cs.store.Jobs[i].State.RunClaimExpiresAtMS = cronInt64Ptr(now + runClaimTTL.Milliseconds())
		}
	}

	if err := cs.saveStoreUnsafe(); err != nil {
		log.Printf("[cron] failed to save store: %v", err)
	}

	cs.mu.Unlock()

	// Execute jobs outside lock.
	for _, jobID := range dueJobIDs {
		cs.parallelWaitGroup.Add(1)
		go func(id string) {
			defer cs.parallelWaitGroup.Done()
			cs.parallelSem <- struct{}{}
			defer func() { <-cs.parallelSem }()
			cs.executeJobByID(id)
		}(jobID)
	}
}

func (cs *CronService) executeJobByID(jobID string) {
	startTime := time.Now().UnixMilli()

	cs.mu.RLock()
	var callbackJob *CronJob
	for i := range cs.store.Jobs {
		job := &cs.store.Jobs[i]
		if job.ID == jobID {
			jobCopy := *job
			callbackJob = &jobCopy
			break
		}
	}
	cs.mu.RUnlock()

	if callbackJob == nil {
		log.Printf("[cron] job %s not found, skipping", jobID)
		return
	}

	// Log job execution start
	log.Printf("[cron] ▶ executing job '%s' (id: %s, schedule: %s, channel: %s)",
		callbackJob.Name, jobID, callbackJob.Schedule.Kind, callbackJob.Payload.Channel)

	var err error
	var output string
	if cs.onJob != nil {
		output, err = cs.onJob(callbackJob)
	}

	execDuration := time.Now().UnixMilli() - startTime

	// Now acquire lock to update state
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var job *CronJob
	for i := range cs.store.Jobs {
		if cs.store.Jobs[i].ID == jobID {
			job = &cs.store.Jobs[i]
			break
		}
	}
	if job == nil {
		log.Printf("[cron] job %s disappeared before state update", jobID)
		return
	}

	job.State.LastRunAtMS = &startTime
	job.State.RunningAtMS = nil
	job.State.RunClaimExpiresAtMS = nil
	job.State.LastDurationMS = execDuration
	job.UpdatedAtMS = time.Now().UnixMilli()

	if err != nil {
		job.State.LastStatus = "error"
		job.State.LastError = err.Error()
		log.Printf("[cron] ✗ job '%s' failed after %dms: %v", job.Name, execDuration, err)
	} else {
		job.State.LastStatus = "ok"
		job.State.LastError = ""
	}
	if outputPath, outputErr := cs.saveJobOutput(callbackJob, output, err, startTime, execDuration); outputErr != nil {
		log.Printf("[cron] failed to save output for job '%s': %v", job.Name, outputErr)
	} else {
		job.State.LastOutputPath = outputPath
	}

	// Compute next run time
	var nextRunStr string
	if job.Schedule.Kind == "at" {
		if job.DeleteAfterRun {
			cs.removeJobUnsafe(job.ID)
			nextRunStr = "(deleted)"
		} else {
			job.Enabled = false
			job.State.NextRunAtMS = nil
			nextRunStr = "(disabled)"
		}
	} else {
		nextRun := cs.computeNextRun(&job.Schedule, time.Now().UnixMilli())
		job.State.NextRunAtMS = nextRun
		if nextRun != nil {
			nextRunStr = time.UnixMilli(*nextRun).Format("2006-01-02 15:04:05")
		} else {
			nextRunStr = "(none)"
		}
	}

	if err == nil {
		log.Printf("[cron] ✓ job '%s' completed in %dms, next run: %s", job.Name, execDuration, nextRunStr)
	}

	if err := cs.saveStoreUnsafe(); err != nil {
		log.Printf("[cron] failed to save store: %v", err)
	}
}

func (cs *CronService) computeNextRun(schedule *CronSchedule, nowMS int64) *int64 {
	switch schedule.Kind {
	case "at":
		if schedule.AtMS != nil && *schedule.AtMS > nowMS {
			return schedule.AtMS
		}
		return nil
	case "every":
		if schedule.EveryMS == nil || *schedule.EveryMS <= 0 {
			return nil
		}
		next := nowMS + *schedule.EveryMS
		return &next
	case "cron":
		if schedule.Expr == "" {
			return nil
		}

		// Use gronx to calculate next run time
		now := time.UnixMilli(nowMS)
		nextTime, err := gronx.NextTickAfter(schedule.Expr, now, false)
		if err != nil {
			log.Printf("[cron] failed to compute next run for expr '%s': %v", schedule.Expr, err)
			return nil
		}

		nextMS := nextTime.UnixMilli()
		return &nextMS
	default:
		log.Printf("[cron] unknown schedule kind '%s'", schedule.Kind)
		return nil
	}
}

// wake up the loop to re-evaluate next wake time immediately (e.g. after add/update/remove jobs)
func (cs *CronService) notify() {
	select {
	case cs.wakeChan <- struct{}{}:
	default:
		// if the channel is full, it means the loop will wake up soon anyway, so we can skip sending
	}
}

func (cs *CronService) recomputeNextRuns() {
	now := time.Now().UnixMilli()
	for i := range cs.store.Jobs {
		job := &cs.store.Jobs[i]
		if job.Enabled && job.State.RunningAtMS == nil {
			job.State.NextRunAtMS = cs.computeNextRun(&job.Schedule, now)
		}
	}
}

func (cs *CronService) recoverStaleRunningJobs(now int64) {
	for i := range cs.store.Jobs {
		job := &cs.store.Jobs[i]
		if job.State.RunningAtMS == nil {
			continue
		}
		if job.State.RunClaimExpiresAtMS != nil && *job.State.RunClaimExpiresAtMS > now {
			continue
		}
		job.State.RunningAtMS = nil
		job.State.RunClaimExpiresAtMS = nil
		job.State.LastStatus = "stale_recovered"
		job.State.LastError = "previous run claim expired before completion"
		if job.Enabled {
			if job.Schedule.Kind == "at" {
				job.State.NextRunAtMS = job.Schedule.AtMS
			} else {
				job.State.NextRunAtMS = cs.computeNextRun(&job.Schedule, now)
			}
		}
	}
}

func (cs *CronService) getNextWakeMS() *int64 {
	var nextWake *int64
	for _, job := range cs.store.Jobs {
		if job.Enabled && job.State.NextRunAtMS != nil {
			if nextWake == nil || *job.State.NextRunAtMS < *nextWake {
				nextWake = job.State.NextRunAtMS
			}
		}
	}
	return nextWake
}

func (cs *CronService) Load() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.loadStore()
}

func (cs *CronService) SetOnJob(handler JobHandler) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.onJob = handler
}

func (cs *CronService) loadStore() error {
	cs.store = &CronStore{
		Version: storeVersion,
		Jobs:    []CronJob{},
	}

	err := withCronFileLock(cs.lockPath, func() error {
		data, err := os.ReadFile(cs.storePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if err := json.Unmarshal(data, cs.store); err != nil {
			return err
		}
		if cs.store.Version == 0 {
			cs.store.Version = 1
		}
		if cs.store.Jobs == nil {
			cs.store.Jobs = []CronJob{}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (cs *CronService) saveStoreUnsafe() error {
	cs.store.Version = storeVersion
	data, err := json.MarshalIndent(cs.store, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return withCronFileLock(cs.lockPath, func() error {
		return fileutil.WriteFileAtomic(cs.storePath, data, 0o600)
	})
}

func (cs *CronService) AddJob(
	name string,
	schedule CronSchedule,
	message string,
	deliver bool,
	deliveryApproved bool,
	channel, to string,
) (*CronJob, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now().UnixMilli()

	// One-time tasks (at) should be deleted after execution
	deleteAfterRun := (schedule.Kind == "at")

	job := CronJob{
		ID:       generateID(),
		Name:     name,
		Enabled:  true,
		Schedule: schedule,
		Payload: CronPayload{
			Kind:             "agent_turn",
			Message:          message,
			Deliver:          deliver,
			DeliveryApproved: deliveryApproved,
			Channel:          channel,
			To:               to,
		},
		State: CronJobState{
			NextRunAtMS: cs.computeNextRun(&schedule, now),
		},
		CreatedAtMS:    now,
		UpdatedAtMS:    now,
		DeleteAfterRun: deleteAfterRun,
	}

	cs.store.Jobs = append(cs.store.Jobs, job)
	if err := cs.saveStoreUnsafe(); err != nil {
		return nil, err
	}

	cs.notify()

	return &job, nil
}

func (cs *CronService) UpdateJob(job *CronJob) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for i := range cs.store.Jobs {
		if cs.store.Jobs[i].ID == job.ID {
			cs.store.Jobs[i] = *job
			cs.store.Jobs[i].UpdatedAtMS = time.Now().UnixMilli()

			cs.notify()

			return cs.saveStoreUnsafe()
		}
	}
	return fmt.Errorf("job not found")
}

func (cs *CronService) RemoveJob(jobID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return cs.removeJobUnsafe(jobID)
}

func (cs *CronService) removeJobUnsafe(jobID string) bool {
	before := len(cs.store.Jobs)
	var jobs []CronJob
	for _, job := range cs.store.Jobs {
		if job.ID != jobID {
			jobs = append(jobs, job)
		}
	}
	cs.store.Jobs = jobs
	removed := len(cs.store.Jobs) < before

	if removed {
		if err := cs.saveStoreUnsafe(); err != nil {
			log.Printf("[cron] failed to save store after remove: %v", err)
		}
	}

	cs.notify()

	return removed
}

func (cs *CronService) EnableJob(jobID string, enabled bool) *CronJob {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for i := range cs.store.Jobs {
		job := &cs.store.Jobs[i]
		if job.ID == jobID {
			job.Enabled = enabled
			job.UpdatedAtMS = time.Now().UnixMilli()

			if enabled {
				job.State.NextRunAtMS = cs.computeNextRun(&job.Schedule, time.Now().UnixMilli())
			} else {
				job.State.NextRunAtMS = nil
			}

			if err := cs.saveStoreUnsafe(); err != nil {
				log.Printf("[cron] failed to save store after enable: %v", err)
			}

			cs.notify()

			return job
		}
	}

	return nil
}

func (cs *CronService) ListJobs(includeDisabled bool) []CronJob {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if includeDisabled {
		return append([]CronJob(nil), cs.store.Jobs...)
	}

	var enabled []CronJob
	for _, job := range cs.store.Jobs {
		if job.Enabled {
			enabled = append(enabled, job)
		}
	}

	return enabled
}

func (cs *CronService) Status() map[string]any {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var enabledCount int
	var runningCount int
	for _, job := range cs.store.Jobs {
		if job.Enabled {
			enabledCount++
		}
		if job.State.RunningAtMS != nil {
			runningCount++
		}
	}

	return map[string]any{
		"enabled":              cs.running,
		"jobs":                 len(cs.store.Jobs),
		"enabledJobs":          enabledCount,
		"runningJobs":          runningCount,
		"nextWakeAtMS":         cs.getNextWakeMS(),
		"cronDir":              cs.cronDir,
		"outputDir":            cs.outputDir,
		"heartbeatPath":        cs.heartbeatPath,
		"lastSuccessPath":      cs.lastSuccessPath,
		"lastHeartbeatModTime": fileModUnixMS(cs.heartbeatPath),
		"lastSuccessfulTickMS": fileModUnixMS(cs.lastSuccessPath),
		"parallelWorkers":      cap(cs.parallelSem),
	}
}

func (cs *CronService) saveJobOutput(job *CronJob, output string, runErr error, startedAtMS, durationMS int64) (string, error) {
	if job == nil {
		return "", nil
	}
	jobID := sanitizePathComponent(job.ID)
	if jobID == "" {
		return "", fmt.Errorf("invalid job id %q", job.ID)
	}
	dir := filepath.Join(cs.outputDir, jobID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	status := "ok"
	errText := ""
	if runErr != nil {
		status = "error"
		errText = runErr.Error()
	}
	ts := time.UnixMilli(startedAtMS).Format("20060102-150405.000")
	path := filepath.Join(dir, ts+".md")
	var b strings.Builder
	b.WriteString("# Cron Job Output\n\n")
	b.WriteString(fmt.Sprintf("- Job: %s\n", job.Name))
	b.WriteString(fmt.Sprintf("- ID: %s\n", job.ID))
	b.WriteString(fmt.Sprintf("- Status: %s\n", status))
	b.WriteString(fmt.Sprintf("- Started: %s\n", time.UnixMilli(startedAtMS).Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Duration: %dms\n", durationMS))
	if errText != "" {
		b.WriteString(fmt.Sprintf("- Error: %s\n", errText))
	}
	b.WriteString("\n## Output\n\n")
	if strings.TrimSpace(output) == "" {
		b.WriteString("(empty)\n")
	} else {
		b.WriteString(output)
		if !strings.HasSuffix(output, "\n") {
			b.WriteString("\n")
		}
	}
	return path, fileutil.WriteFileAtomic(path, []byte(b.String()), 0o600)
}

func IsSilentResponse(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	normalized := strings.ToUpper(strings.Join(strings.Fields(trimmed), " "))
	switch normalized {
	case "[SILENT]", "SILENT", "NO_REPLY", "NO REPLY":
		return true
	}
	lines := strings.Split(trimmed, "\n")
	first := strings.ToUpper(strings.Join(strings.Fields(lines[0]), " "))
	last := strings.ToUpper(strings.Join(strings.Fields(lines[len(lines)-1]), " "))
	return first == "[SILENT]" || last == "[SILENT]" || strings.HasPrefix(strings.ToUpper(trimmed), "[SILENT]")
}

func sanitizePathComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\`) {
		return ""
	}
	if filepath.IsAbs(value) {
		return ""
	}
	return value
}

func (cs *CronService) writeHeartbeat(path string) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		log.Printf("[cron] failed to create heartbeat dir: %v", err)
		return
	}
	now := []byte(fmt.Sprintf("%d\n", time.Now().UnixMilli()))
	if err := fileutil.WriteFileAtomic(path, now, 0o600); err != nil {
		log.Printf("[cron] failed to write heartbeat: %v", err)
	}
}

func fileModUnixMS(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixMilli()
}

func cronInt64Ptr(v int64) *int64 {
	return &v
}

func generateID() string {
	// Use crypto/rand for better uniqueness under concurrent access
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
