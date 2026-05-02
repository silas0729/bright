package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleLearnerCaptcha(c *gin.Context) {
	challenge, err := s.service.IssueCaptcha(c.DefaultQuery("scene", "learner_login"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, challenge)
}

func (s *Server) handleAdminCaptcha(c *gin.Context) {
	challenge, err := s.service.IssueCaptcha("admin_login")
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, challenge)
}
