package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminSetupStatus(c *gin.Context) {
	status, err := s.service.GetAdminSetupStatus(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Server) handleAdminSetupBootstrap(c *gin.Context) {
	var input domain.InitializeSuperAdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	admin, err := s.service.InitializeSuperAdmin(c.Request.Context(), input)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "already been initialized") {
			status = http.StatusConflict
		}
		writeError(c, status, err)
		return
	}

	session, err := s.auth.IssueAdminSession(admin)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, session)
}
