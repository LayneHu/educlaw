package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleHealth returns process liveness.
func (s *Server) HandleHealth(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	c.JSON(http.StatusOK, s.health.HealthStatus())
}

// HandleReady returns dependency readiness.
func (s *Server) HandleReady(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
		return
	}
	resp, ok := s.health.ReadyStatus()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}
