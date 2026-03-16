package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pingjie/educlaw/pkg/storage"
)

// HandleClassReport returns class-level learning analytics for a teacher.
func (s *Server) HandleClassReport(c *gin.Context) {
	teacherID := c.Param("id")
	if teacherID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "teacher id required"})
		return
	}

	teacher, err := storage.GetActor(s.db, teacherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if teacher == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "teacher not found"})
		return
	}

	// Get all students for class analysis
	students, err := storage.ListActors(s.db, "student")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Aggregate knowledge stats across all students
	subjectStats := make(map[string]map[string]*struct {
		kpName       string
		totalCorrect int
		totalAttempt int
		studentCount int
	})

	studentSummaries := make([]map[string]any, 0, len(students))
	for _, student := range students {
		states, _ := storage.GetKnowledgeStates(s.db, student.ID)
		totalMastery := 0
		count := 0
		for _, ks := range states {
			totalMastery += ks.MasteryPercent()
			count++

			if _, ok := subjectStats[ks.Subject]; !ok {
				subjectStats[ks.Subject] = make(map[string]*struct {
					kpName       string
					totalCorrect int
					totalAttempt int
					studentCount int
				})
			}
			if _, ok := subjectStats[ks.Subject][ks.KpID]; !ok {
				subjectStats[ks.Subject][ks.KpID] = &struct {
					kpName       string
					totalCorrect int
					totalAttempt int
					studentCount int
				}{kpName: ks.KpName}
			}
			stat := subjectStats[ks.Subject][ks.KpID]
			stat.totalCorrect += ks.CorrectCount
			stat.totalAttempt += ks.TotalCount
			stat.studentCount++
		}

		avgMastery := 0
		if count > 0 {
			avgMastery = totalMastery / count
		}
		studentSummaries = append(studentSummaries, map[string]any{
			"id":          student.ID,
			"name":        student.Name,
			"avg_mastery": avgMastery,
		})
	}

	// Format subject stats
	classStats := make(map[string][]map[string]any)
	for subject, kps := range subjectStats {
		for kpID, stat := range kps {
			mastery := 0
			if stat.totalAttempt > 0 {
				mastery = stat.totalCorrect * 100 / stat.totalAttempt
			}
			classStats[subject] = append(classStats[subject], map[string]any{
				"kp_id":         kpID,
				"kp_name":       stat.kpName,
				"class_mastery": mastery,
				"student_count": stat.studentCount,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"teacher_id":       teacherID,
		"teacher":          teacher.Name,
		"student_count":    len(students),
		"students":         studentSummaries,
		"class_stats":      classStats,
	})
}
