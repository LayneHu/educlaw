package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// KnowledgeState represents the mastery level of a knowledge point.
type KnowledgeState struct {
	ID           string
	StudentID    string
	Subject      string
	KpID         string
	KpName       string
	CorrectCount int
	TotalCount   int
	LastSeen     time.Time
}

// MasteryPercent returns the mastery percentage (0-100).
func (ks *KnowledgeState) MasteryPercent() int {
	if ks.TotalCount == 0 {
		return 0
	}
	return int(float64(ks.CorrectCount) / float64(ks.TotalCount) * 100)
}

// UpsertKnowledge inserts or updates a knowledge state record.
func UpsertKnowledge(db *sql.DB, ks KnowledgeState) error {
	if ks.ID == "" {
		ks.ID = uuid.New().String()
	}
	_, err := db.Exec(
		`INSERT INTO knowledge_states (id, student_id, subject, kp_id, kp_name, correct_count, total_count, last_seen)
		 VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(student_id, subject, kp_id) DO UPDATE SET
		   kp_name = excluded.kp_name,
		   correct_count = correct_count + excluded.correct_count,
		   total_count = total_count + excluded.total_count,
		   last_seen = CURRENT_TIMESTAMP`,
		ks.ID, ks.StudentID, ks.Subject, ks.KpID, ks.KpName, ks.CorrectCount, ks.TotalCount,
	)
	if err != nil {
		return fmt.Errorf("upserting knowledge state: %w", err)
	}
	return nil
}

// GetKnowledgeStates retrieves all knowledge states for a student.
func GetKnowledgeStates(db *sql.DB, studentID string) ([]KnowledgeState, error) {
	rows, err := db.Query(
		`SELECT id, student_id, subject, kp_id, kp_name, correct_count, total_count, last_seen
		 FROM knowledge_states
		 WHERE student_id = ?
		 ORDER BY subject, kp_name`,
		studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying knowledge states: %w", err)
	}
	defer rows.Close()

	var states []KnowledgeState
	for rows.Next() {
		var ks KnowledgeState
		if err := rows.Scan(&ks.ID, &ks.StudentID, &ks.Subject, &ks.KpID, &ks.KpName,
			&ks.CorrectCount, &ks.TotalCount, &ks.LastSeen); err != nil {
			return nil, fmt.Errorf("scanning knowledge state: %w", err)
		}
		states = append(states, ks)
	}
	return states, rows.Err()
}
