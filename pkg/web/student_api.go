package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pingjie/educlaw/pkg/bus"
	"github.com/pingjie/educlaw/pkg/commands"
	"github.com/pingjie/educlaw/pkg/storage"
)

// ChatRequest represents a chat API request.
type ChatRequest struct {
	ActorID   string `json:"actor_id"`
	ActorType string `json:"actor_type"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// ChatResponse represents a chat API response.
type ChatResponse struct {
	SessionID string `json:"session_id"`
}

// HandleChat processes incoming chat messages and kicks off the agent loop.
func (s *Server) HandleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ActorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "actor_id is required"})
		return
	}
	if req.ActorType == "" {
		req.ActorType = "student"
	}
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	// Generate session ID if not provided
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	msg := bus.InboundMessage{
		ActorID:   req.ActorID,
		ActorType: req.ActorType,
		SessionID: sessionID,
		Content:   req.Content,
	}

	if s.commands != nil {
		result := s.commands.Execute(context.Background(), s.commandRequest(msg))
		if result.Handled {
			c.JSON(http.StatusOK, ChatResponse{SessionID: sessionID})
			return
		}
	}

	// Process asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.agentLoop.Process(ctx, msg); err != nil {
			log.Printf("agent loop error for session %s: %v", sessionID, err)
			s.msgBus.Publish(bus.OutboundMessage{
				SessionID:   sessionID,
				ActorID:     req.ActorID,
				Content:     fmt.Sprintf("处理请求时出错: %v", err),
				ContentType: "error",
				Done:        true,
			})
		}
	}()

	c.JSON(http.StatusOK, ChatResponse{SessionID: sessionID})
}

func (s *Server) commandRequest(msg bus.InboundMessage) commands.Request {
	return commands.Request{
		Text:      msg.Content,
		SessionID: msg.SessionID,
		ActorID:   msg.ActorID,
		ActorType: msg.ActorType,
		Reply: func(content string) error {
			s.msgBus.Publish(bus.OutboundMessage{
				SessionID:   msg.SessionID,
				ActorID:     msg.ActorID,
				Content:     content,
				ContentType: "text",
				Done:        false,
			})
			s.msgBus.Publish(bus.OutboundMessage{
				SessionID:   msg.SessionID,
				ActorID:     msg.ActorID,
				Content:     "",
				ContentType: "text",
				Done:        true,
			})
			return nil
		},
	}
}

// HandleStream serves SSE stream for a given session.
func (s *Server) HandleStream(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Subscribe to session messages
	ch := s.msgBus.Subscribe(sessionID)
	defer s.msgBus.Unsubscribe(sessionID, ch)

	clientGone := c.Request.Context().Done()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case msg, ok := <-ch:
			if !ok {
				return false
			}

			switch msg.ContentType {
			case "text":
				// Send text token
				data, _ := json.Marshal(map[string]any{
					"type":    "text",
					"content": msg.Content,
					"done":    msg.Done,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
			case "rendered":
				// Send rendered content
				data, _ := json.Marshal(map[string]any{
					"type":    "rendered",
					"content": msg.Content,
					"done":    msg.Done,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
			case "tool_call":
				// Forward tool event as-is
				data, _ := json.Marshal(map[string]any{
					"type":    "tool_call",
					"content": msg.Content,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
			case "error":
				data, _ := json.Marshal(map[string]any{
					"type":    "error",
					"content": msg.Content,
					"done":    true,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
			}

			if msg.Done {
				// Send final done event
				data, _ := json.Marshal(map[string]any{
					"type": "done",
					"done": true,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				return false
			}
			return true
		case <-time.After(30 * time.Second):
			// Heartbeat
			fmt.Fprintf(w, ": heartbeat\n\n")
			return true
		}
	})
}

// HandleStudentSummary returns a student's knowledge summary.
func (s *Server) HandleStudentSummary(c *gin.Context) {
	studentID := c.Param("id")
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
		return
	}

	states, err := storage.GetKnowledgeStates(s.db, studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Group by subject
	bySubject := make(map[string][]map[string]any)
	for _, ks := range states {
		item := map[string]any{
			"kp_id":         ks.KpID,
			"kp_name":       ks.KpName,
			"correct_count": ks.CorrectCount,
			"total_count":   ks.TotalCount,
			"mastery":       ks.MasteryPercent(),
		}
		bySubject[ks.Subject] = append(bySubject[ks.Subject], item)
	}

	c.JSON(http.StatusOK, gin.H{
		"student_id": studentID,
		"knowledge":  bySubject,
	})
}

// HandleOnboard handles new actor registration.
func (s *Server) HandleOnboard(c *gin.Context) {
	var req struct {
		ActorType string `json:"actor_type"`
		Name      string `json:"name"`
		Grade     string `json:"grade"`
		Subject   string `json:"subject"`
		FamilyID  string `json:"family_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	if err := storage.SaveActor(s.db, id, req.ActorType, req.Name, req.Grade, req.Subject, req.FamilyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Initialize workspace
	var actorDir string
	switch req.ActorType {
	case "student":
		actorDir = s.wm.StudentDir(id)
	case "family", "parent":
		actorDir = s.wm.FamilyDir(id)
	case "teacher":
		actorDir = s.wm.TeacherDir(id)
	}

	if actorDir != "" {
		_ = s.wm.WriteFile(actorDir, "PROFILE.md", fmt.Sprintf("# %s\n\nName: %s\nType: %s\nGrade: %s\n",
			req.Name, req.Name, req.ActorType, req.Grade))
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "name": req.Name, "actor_type": req.ActorType})
}

// HandleListActors lists all actors of the given type.
func (s *Server) HandleListActors(c *gin.Context) {
	actorType := c.Param("type")
	actors, err := storage.ListActors(s.db, actorType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]map[string]any, 0, len(actors))
	for _, a := range actors {
		result = append(result, map[string]any{
			"id":         a.ID,
			"actor_type": a.ActorType,
			"name":       a.Name,
			"grade":      a.Grade,
			"subject":    a.Subject,
			"family_id":  a.FamilyID,
		})
	}
	c.JSON(http.StatusOK, gin.H{"actors": result})
}
