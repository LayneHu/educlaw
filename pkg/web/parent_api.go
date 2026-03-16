package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pingjie/educlaw/pkg/storage"
)

// HandleParentReport returns a parent-facing report of their child's progress.
func (s *Server) HandleParentReport(c *gin.Context) {
	parentID := c.Param("id")
	if parentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parent id required"})
		return
	}

	// Get children linked to this family/parent
	actor, err := storage.GetActor(s.db, parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if actor == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "parent not found"})
		return
	}

	// Find students associated with this family
	students, err := storage.ListActors(s.db, "student")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter students belonging to this family
	var familyStudents []map[string]any
	for _, student := range students {
		if student.FamilyID == parentID {
			states, _ := storage.GetKnowledgeStates(s.db, student.ID)
			bySubject := make(map[string][]map[string]any)
			for _, ks := range states {
				item := map[string]any{
					"kp_name": ks.KpName,
					"mastery": ks.MasteryPercent(),
				}
				bySubject[ks.Subject] = append(bySubject[ks.Subject], item)
			}
			familyStudents = append(familyStudents, map[string]any{
				"id":        student.ID,
				"name":      student.Name,
				"grade":     student.Grade,
				"knowledge": bySubject,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"parent_id": parentID,
		"parent":    actor.Name,
		"students":  familyStudents,
	})
}
