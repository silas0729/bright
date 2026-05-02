package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminPlans(c *gin.Context) {
	items, err := s.service.ListPlans(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleUpdatePlan(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var input domain.UpdatePlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdatePlan(c.Request.Context(), uint(id), input)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			status = http.StatusNotFound
		}
		writeError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleDeletePlan(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.service.DeletePlan(c.Request.Context(), uint(id)); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			status = http.StatusNotFound
		}
		writeError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
