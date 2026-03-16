package health

import (
	"sync"
	"time"
)

// Check captures the latest evaluation of a readiness check.
type Check struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// StatusResponse is the JSON payload returned by health endpoints.
type StatusResponse struct {
	Status string           `json:"status"`
	Uptime string           `json:"uptime"`
	Checks map[string]Check `json:"checks,omitempty"`
}

// CheckFunc evaluates a runtime dependency.
type CheckFunc func() (bool, string)

// Manager tracks readiness and runtime health checks for the current process.
type Manager struct {
	mu        sync.RWMutex
	ready     bool
	checks    map[string]CheckFunc
	startTime time.Time
}

// NewManager creates a new health manager.
func NewManager() *Manager {
	return &Manager{
		checks:    make(map[string]CheckFunc),
		startTime: time.Now(),
	}
}

// SetReady updates the service readiness flag.
func (m *Manager) SetReady(ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = ready
}

// RegisterCheck adds or replaces a named readiness check.
func (m *Manager) RegisterCheck(name string, fn CheckFunc) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks[name] = fn
}

// HealthStatus returns the liveness payload. Liveness does not fail on checks.
func (m *Manager) HealthStatus() StatusResponse {
	return StatusResponse{
		Status: "ok",
		Uptime: time.Since(m.startTime).String(),
	}
}

// ReadyStatus returns the readiness payload and whether the service is ready.
func (m *Manager) ReadyStatus() (StatusResponse, bool) {
	m.mu.RLock()
	ready := m.ready
	checkFns := make(map[string]CheckFunc, len(m.checks))
	for name, fn := range m.checks {
		checkFns[name] = fn
	}
	m.mu.RUnlock()

	checks := make(map[string]Check, len(checkFns))
	allOK := ready
	now := time.Now()
	for name, fn := range checkFns {
		ok, msg := fn()
		checks[name] = Check{
			Name:      name,
			Status:    statusString(ok),
			Message:   msg,
			Timestamp: now,
		}
		if !ok {
			allOK = false
		}
	}

	resp := StatusResponse{
		Status: "ready",
		Uptime: time.Since(m.startTime).String(),
		Checks: checks,
	}
	if !allOK {
		resp.Status = "not ready"
	}
	return resp, allOK
}

func statusString(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}
