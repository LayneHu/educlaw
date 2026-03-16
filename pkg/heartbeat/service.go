package heartbeat

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pingjie/educlaw/pkg/bus"
	"github.com/pingjie/educlaw/pkg/config"
	"github.com/pingjie/educlaw/pkg/storage"
)

const (
	minIntervalMinutes     = 5
	defaultIntervalMinutes = 30
)

// Handler is the function called for each active session during a heartbeat.
// It returns an error if processing fails.
type Handler func(ctx context.Context, sessionID, actorID, actorType string) error

// Service manages periodic heartbeat checks for connected students.
type Service struct {
	db       *sql.DB
	msgBus   *bus.MessageBus
	cfg      config.HeartbeatConfig
	handler  Handler
	mu       sync.Mutex
	stopChan chan struct{}
}

// NewService creates a new heartbeat Service.
func NewService(db *sql.DB, msgBus *bus.MessageBus, cfg config.HeartbeatConfig) *Service {
	return &Service{
		db:     db,
		msgBus: msgBus,
		cfg:    cfg,
	}
}

// SetHandler sets the function to call for each active session.
func (s *Service) SetHandler(h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = h
}

// Start begins the heartbeat service. Safe to call multiple times.
func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled {
		log.Printf("[heartbeat] service disabled")
		return
	}
	if s.stopChan != nil {
		return // already running
	}

	intervalMinutes := s.cfg.IntervalMinutes
	if intervalMinutes < minIntervalMinutes {
		intervalMinutes = defaultIntervalMinutes
	}

	s.stopChan = make(chan struct{})
	go s.runLoop(s.stopChan, time.Duration(intervalMinutes)*time.Minute)
	log.Printf("[heartbeat] started (interval: %d min)", intervalMinutes)
}

// Stop gracefully stops the heartbeat service.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = nil
	}
}

func (s *Service) runLoop(stop chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.execute()
		}
	}
}

// execute runs a heartbeat check for all recently active sessions that have
// live SSE subscribers.
func (s *Service) execute() {
	s.mu.Lock()
	handler := s.handler
	s.mu.Unlock()

	if handler == nil {
		return
	}

	// Find sessions updated in the last 2 hours
	sessions, err := storage.GetRecentlyActiveSessions(s.db, 2)
	if err != nil {
		log.Printf("[heartbeat] error fetching active sessions: %v", err)
		return
	}

	for _, sess := range sessions {
		// Only push to sessions that have active SSE subscribers
		if !s.msgBus.HasSubscribers(sess.ID) {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		if err := handler(ctx, sess.ID, sess.ActorID, sess.ActorType); err != nil {
			log.Printf("[heartbeat] session %s error: %v", sess.ID, err)
		}
		cancel()
	}
}

// BuildHeartbeatPrompt returns the prompt used for heartbeat checks.
func BuildHeartbeatPrompt(actorType string) string {
	now := time.Now().Format("2006-01-02 15:04")
	switch actorType {
	case "student":
		return fmt.Sprintf(`[heartbeat check at %s]
你好！这是一次主动学习提醒。请检查学生的学习记录，考虑：
1. 是否有需要复习的知识点（间隔重复）？
2. 是否有未完成的学习计划？
3. 根据学生的兴趣，是否有值得探索的新话题？

如果有值得提醒的内容，用简短友好的语气告知学生（不超过3条）。
如果没有需要特别提醒的，只需回复一句简短的鼓励话语即可。`, now)
	case "parent", "family":
		return fmt.Sprintf(`[heartbeat check at %s]
这是一次定期学习进度提醒。请检查关联学生的最近学习情况，如有重要进展或需要关注的地方，
用简短的语言告知家长。如无特别情况，可以发送一条简短的积极反馈。`, now)
	default:
		return fmt.Sprintf(`[heartbeat check at %s]
请检查是否有需要处理的待办事项。`, now)
	}
}
