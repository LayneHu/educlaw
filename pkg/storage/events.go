package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// LearningEvent represents a single learning interaction.
type LearningEvent struct {
	ID        string
	StudentID string
	Subject   string
	KpID      string
	KpName    string
	IsCorrect bool
	Note      string
	CreatedAt time.Time
}

// RecordEvent saves a learning event to the database.
func RecordEvent(db *sql.DB, event LearningEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	isCorrect := 0
	if event.IsCorrect {
		isCorrect = 1
	}
	_, err := db.Exec(
		`INSERT INTO learning_events (id, student_id, subject, kp_id, kp_name, is_correct, note)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.StudentID, event.Subject, event.KpID, event.KpName, isCorrect, event.Note,
	)
	if err != nil {
		return fmt.Errorf("recording event: %w", err)
	}
	return nil
}

// GetRecentEvents retrieves recent learning events for a student.
func GetRecentEvents(db *sql.DB, studentID string, limit int) ([]LearningEvent, error) {
	rows, err := db.Query(
		`SELECT id, student_id, subject, kp_id, kp_name, is_correct, note, created_at
		 FROM learning_events
		 WHERE student_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		studentID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}
	defer rows.Close()

	var events []LearningEvent
	for rows.Next() {
		var e LearningEvent
		var isCorrect int
		if err := rows.Scan(&e.ID, &e.StudentID, &e.Subject, &e.KpID, &e.KpName, &isCorrect, &e.Note, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		e.IsCorrect = isCorrect == 1
		events = append(events, e)
	}
	return events, rows.Err()
}
