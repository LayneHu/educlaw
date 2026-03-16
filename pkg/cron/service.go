package cron

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adhocore/gronx"
)

// Schedule kinds.
const (
	KindAt    = "at"    // one-time, fires at AtMS
	KindEvery = "every" // repeating interval, EveryMS milliseconds
	KindCron  = "cron"  // cron expression, Expr field
)

// Schedule defines when a job runs.
type Schedule struct {
	Kind    string `json:"kind"`
	AtMS    *int64 `json:"atMs,omitempty"`    // unix ms, for "at"
	EveryMS *int64 `json:"everyMs,omitempty"` // duration ms, for "every"
	Expr    string `json:"expr,omitempty"`    // cron expression, for "cron"
}

// Payload carries the message to deliver when a job fires.
type Payload struct {
	Message   string `json:"message"`
	ActorID   string `json:"actor_id"`
	ActorType string `json:"actor_type"`
}

// JobState tracks execution state.
type JobState struct {
	NextRunAtMS *int64 `json:"nextRunAtMs,omitempty"`
	LastRunAtMS *int64 `json:"lastRunAtMs,omitempty"`
	LastStatus  string `json:"lastStatus,omitempty"`
	LastError   string `json:"lastError,omitempty"`
}

// Job is a single scheduled task.
type Job struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	Schedule       Schedule `json:"schedule"`
	Payload        Payload  `json:"payload"`
	State          JobState `json:"state"`
	CreatedAtMS    int64    `json:"createdAtMs"`
	UpdatedAtMS    int64    `json:"updatedAtMs"`
	DeleteAfterRun bool     `json:"deleteAfterRun"`
}

// store is the on-disk persisted list of jobs.
type store struct {
	Version int   `json:"version"`
	Jobs    []Job `json:"jobs"`
}

// JobHandler is called when a job fires. Returns an error on failure.
type JobHandler func(job *Job) error

// Service manages scheduled jobs with persistence.
type Service struct {
	storePath string
	data      *store
	handler   JobHandler
	mu        sync.RWMutex
	running   bool
	stopChan  chan struct{}
	gronx     *gronx.Gronx
}

// NewService creates a new cron Service. The store is loaded from storePath
// (created on first save if it doesn't exist).
func NewService(storePath string) *Service {
	svc := &Service{
		storePath: storePath,
		gronx:     gronx.New(),
	}
	svc.loadStore()
	return svc
}

// SetHandler sets the function called when a job fires. Safe to call before Start.
func (s *Service) SetHandler(h JobHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = h
}

// Start begins the scheduler loop. Safe to call multiple times.
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}
	if err := s.loadStore(); err != nil {
		return fmt.Errorf("loading cron store: %w", err)
	}
	s.recomputeNextRuns()
	_ = s.saveStoreUnsafe()

	s.stopChan = make(chan struct{})
	s.running = true
	go s.runLoop(s.stopChan)
	log.Printf("[cron] started (%d jobs)", len(s.data.Jobs))
	return nil
}

// Stop gracefully stops the scheduler loop.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopChan)
	s.stopChan = nil
}

func (s *Service) runLoop(stop chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.checkJobs()
		}
	}
}

func (s *Service) checkJobs() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	now := time.Now().UnixMilli()
	var due []string
	for i := range s.data.Jobs {
		j := &s.data.Jobs[i]
		if j.Enabled && j.State.NextRunAtMS != nil && *j.State.NextRunAtMS <= now {
			due = append(due, j.ID)
			j.State.NextRunAtMS = nil // clear to prevent double-fire
		}
	}
	if len(due) > 0 {
		_ = s.saveStoreUnsafe()
	}
	s.mu.Unlock()

	for _, id := range due {
		s.executeByID(id)
	}
}

func (s *Service) executeByID(id string) {
	s.mu.RLock()
	var jobCopy *Job
	for i := range s.data.Jobs {
		if s.data.Jobs[i].ID == id {
			cp := s.data.Jobs[i]
			jobCopy = &cp
			break
		}
	}
	s.mu.RUnlock()
	if jobCopy == nil {
		return
	}

	start := time.Now().UnixMilli()
	log.Printf("[cron] ▶ job '%s' (%s) actor=%s", jobCopy.Name, jobCopy.Schedule.Kind, jobCopy.Payload.ActorID)

	var execErr error
	if s.handler != nil {
		execErr = s.handler(jobCopy)
	}
	elapsed := time.Now().UnixMilli() - start

	s.mu.Lock()
	defer s.mu.Unlock()

	var j *Job
	for i := range s.data.Jobs {
		if s.data.Jobs[i].ID == id {
			j = &s.data.Jobs[i]
			break
		}
	}
	if j == nil {
		return
	}

	j.State.LastRunAtMS = &start
	j.UpdatedAtMS = time.Now().UnixMilli()

	if execErr != nil {
		j.State.LastStatus = "error"
		j.State.LastError = execErr.Error()
		log.Printf("[cron] ✗ job '%s' failed in %dms: %v", j.Name, elapsed, execErr)
	} else {
		j.State.LastStatus = "ok"
		j.State.LastError = ""
		log.Printf("[cron] ✓ job '%s' done in %dms", j.Name, elapsed)
	}

	if j.Schedule.Kind == KindAt {
		if j.DeleteAfterRun {
			s.removeUnsafe(id)
		} else {
			j.Enabled = false
		}
	} else {
		next := s.computeNext(&j.Schedule, time.Now().UnixMilli())
		j.State.NextRunAtMS = next
	}

	_ = s.saveStoreUnsafe()
}

// AddJob creates and persists a new scheduled job.
func (s *Service) AddJob(name string, schedule Schedule, payload Payload) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	j := Job{
		ID:             generateID(),
		Name:           name,
		Enabled:        true,
		Schedule:       schedule,
		Payload:        payload,
		CreatedAtMS:    now,
		UpdatedAtMS:    now,
		DeleteAfterRun: schedule.Kind == KindAt,
	}
	j.State.NextRunAtMS = s.computeNext(&schedule, now)

	s.data.Jobs = append(s.data.Jobs, j)
	if err := s.saveStoreUnsafe(); err != nil {
		return nil, err
	}
	log.Printf("[cron] added job '%s' (id=%s, kind=%s)", name, j.ID, schedule.Kind)
	return &j, nil
}

// RemoveJob deletes a job by ID. Returns true if removed.
func (s *Service) RemoveJob(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := s.removeUnsafe(id)
	if removed {
		_ = s.saveStoreUnsafe()
	}
	return removed
}

func (s *Service) removeUnsafe(id string) bool {
	before := len(s.data.Jobs)
	jobs := s.data.Jobs[:0]
	for _, j := range s.data.Jobs {
		if j.ID != id {
			jobs = append(jobs, j)
		}
	}
	s.data.Jobs = jobs
	return len(s.data.Jobs) < before
}

// ListJobs returns all jobs, or only enabled ones if includeDisabled is false.
func (s *Service) ListJobs(includeDisabled bool) []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if includeDisabled {
		out := make([]Job, len(s.data.Jobs))
		copy(out, s.data.Jobs)
		return out
	}
	var out []Job
	for _, j := range s.data.Jobs {
		if j.Enabled {
			out = append(out, j)
		}
	}
	return out
}

// ListJobsForActor returns all jobs for a specific actor.
func (s *Service) ListJobsForActor(actorID string) []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Job
	for _, j := range s.data.Jobs {
		if j.Payload.ActorID == actorID {
			out = append(out, j)
		}
	}
	return out
}

func (s *Service) computeNext(sch *Schedule, nowMS int64) *int64 {
	switch sch.Kind {
	case KindAt:
		if sch.AtMS != nil && *sch.AtMS > nowMS {
			return sch.AtMS
		}
		return nil
	case KindEvery:
		if sch.EveryMS == nil || *sch.EveryMS <= 0 {
			return nil
		}
		next := nowMS + *sch.EveryMS
		return &next
	case KindCron:
		if sch.Expr == "" {
			return nil
		}
		now := time.UnixMilli(nowMS)
		next, err := gronx.NextTickAfter(sch.Expr, now, false)
		if err != nil {
			log.Printf("[cron] invalid cron expr '%s': %v", sch.Expr, err)
			return nil
		}
		ms := next.UnixMilli()
		return &ms
	}
	return nil
}

func (s *Service) recomputeNextRuns() {
	now := time.Now().UnixMilli()
	for i := range s.data.Jobs {
		j := &s.data.Jobs[i]
		if j.Enabled {
			j.State.NextRunAtMS = s.computeNext(&j.Schedule, now)
		}
	}
}

func (s *Service) loadStore() error {
	s.data = &store{Version: 1, Jobs: []Job{}}
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, s.data)
}

func (s *Service) saveStoreUnsafe() error {
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.storePath, data)
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".cron-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
