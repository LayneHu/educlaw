package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pingjie/educlaw/pkg/llm"
)

// Actor represents a user in the system.
type Actor struct {
	ID        string
	ActorType string
	Name      string
	Grade     string
	Subject   string
	FamilyID  string
	TeacherID string
	CreatedAt time.Time
}

// RenderedContentRow represents a rendered content record in the database.
type RenderedContentRow struct {
	ID          string
	SessionID   string
	ActorID     string
	ContentType string
	Title       string
	Content     string
	CreatedAt   time.Time
}

// GetOrCreateSession retrieves an existing session or creates a new one.
func GetOrCreateSession(db *sql.DB, actorID, actorType string) (string, []llm.Message, error) {
	var sessionID string
	var messagesJSON string

	err := db.QueryRow(
		`SELECT id, messages FROM sessions WHERE actor_id = ? ORDER BY updated_at DESC LIMIT 1`,
		actorID,
	).Scan(&sessionID, &messagesJSON)

	if err == sql.ErrNoRows {
		// Create new session
		sessionID = uuid.New().String()
		_, err = db.Exec(
			`INSERT INTO sessions (id, actor_id, actor_type, messages) VALUES (?, ?, ?, '[]')`,
			sessionID, actorID, actorType,
		)
		if err != nil {
			return "", nil, fmt.Errorf("creating session: %w", err)
		}
		return sessionID, []llm.Message{}, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("querying session: %w", err)
	}

	var messages []llm.Message
	if messagesJSON != "" && messagesJSON != "[]" {
		if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
			// If corrupt, start fresh
			messages = []llm.Message{}
		}
	}

	return sessionID, messages, nil
}

// GetOrCreateSessionByID retrieves a session by its specific ID or creates a new one with that ID.
func GetOrCreateSessionByID(db *sql.DB, sessionID, actorID, actorType string) ([]llm.Message, error) {
	var messagesJSON string
	err := db.QueryRow(
		`SELECT messages FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&messagesJSON)

	if err == sql.ErrNoRows {
		// Create new session with the provided ID
		_, err = db.Exec(
			`INSERT INTO sessions (id, actor_id, actor_type, messages) VALUES (?, ?, ?, '[]')`,
			sessionID, actorID, actorType,
		)
		if err != nil {
			return nil, fmt.Errorf("creating session: %w", err)
		}
		return []llm.Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying session: %w", err)
	}

	var messages []llm.Message
	if messagesJSON != "" && messagesJSON != "[]" {
		if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
			messages = []llm.Message{}
		}
	}
	return messages, nil
}

// ActiveSession holds minimal info about a recently updated session.
type ActiveSession struct {
	ID        string
	ActorID   string
	ActorType string
}

// GetRecentlyActiveSessions returns sessions updated within the last N hours.
func GetRecentlyActiveSessions(db *sql.DB, hours int) ([]ActiveSession, error) {
	rows, err := db.Query(
		`SELECT id, actor_id, actor_type FROM sessions
		 WHERE updated_at >= datetime('now', ? || ' hours')
		 ORDER BY updated_at DESC`,
		fmt.Sprintf("-%d", hours),
	)
	if err != nil {
		return nil, fmt.Errorf("querying recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ActiveSession
	for rows.Next() {
		var s ActiveSession
		if err := rows.Scan(&s.ID, &s.ActorID, &s.ActorType); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// SaveSession persists the current message history for a session.
func SaveSession(db *sql.DB, sessionID string, messages []llm.Message) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("marshaling messages: %w", err)
	}
	_, err = db.Exec(
		`UPDATE sessions SET messages = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		string(data), sessionID,
	)
	if err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
}

// SaveRenderedContent saves a rendered content record and returns its ID.
func SaveRenderedContent(db *sql.DB, sessionID, actorID string, rc RenderedContentRow) (string, error) {
	if rc.ID == "" {
		rc.ID = uuid.New().String()
	}
	_, err := db.Exec(
		`INSERT INTO rendered_contents (id, session_id, actor_id, content_type, title, content)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		rc.ID, sessionID, actorID, rc.ContentType, rc.Title, rc.Content,
	)
	if err != nil {
		return "", fmt.Errorf("saving rendered content: %w", err)
	}
	return rc.ID, nil
}

// SaveActor creates or updates an actor record.
func SaveActor(db *sql.DB, id, actorType, name, grade, subject, familyID, teacherID string) error {
	_, err := db.Exec(
		`INSERT OR REPLACE INTO actors (id, actor_type, name, grade, subject, family_id, teacher_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, actorType, name, grade, subject, familyID, teacherID,
	)
	if err != nil {
		return fmt.Errorf("saving actor: %w", err)
	}
	return nil
}

// GetActor retrieves an actor by ID.
func GetActor(db *sql.DB, id string) (*Actor, error) {
	var a Actor
	err := db.QueryRow(
		`SELECT id, actor_type, name, grade, subject, family_id, COALESCE(teacher_id,''), created_at FROM actors WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.ActorType, &a.Name, &a.Grade, &a.Subject, &a.FamilyID, &a.TeacherID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting actor: %w", err)
	}
	return &a, nil
}

// ListActors lists all actors of a given type.
func ListActors(db *sql.DB, actorType string) ([]Actor, error) {
	rows, err := db.Query(
		`SELECT id, actor_type, name, grade, subject, family_id, COALESCE(teacher_id,''), created_at FROM actors WHERE actor_type = ?`,
		actorType,
	)
	if err != nil {
		return nil, fmt.Errorf("listing actors: %w", err)
	}
	defer rows.Close()

	var actors []Actor
	for rows.Next() {
		var a Actor
		if err := rows.Scan(&a.ID, &a.ActorType, &a.Name, &a.Grade, &a.Subject, &a.FamilyID, &a.TeacherID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning actor: %w", err)
		}
		actors = append(actors, a)
	}
	return actors, rows.Err()
}

// ListActorsByTeacher lists all students assigned to a given teacher.
func ListActorsByTeacher(db *sql.DB, teacherID string) ([]Actor, error) {
	rows, err := db.Query(
		`SELECT id, actor_type, name, grade, subject, family_id, COALESCE(teacher_id,''), created_at
		 FROM actors WHERE actor_type = 'student' AND teacher_id = ?`,
		teacherID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing actors by teacher: %w", err)
	}
	defer rows.Close()

	var actors []Actor
	for rows.Next() {
		var a Actor
		if err := rows.Scan(&a.ID, &a.ActorType, &a.Name, &a.Grade, &a.Subject, &a.FamilyID, &a.TeacherID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning actor: %w", err)
		}
		actors = append(actors, a)
	}
	return actors, rows.Err()
}
