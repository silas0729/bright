package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminLearners(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListLearnerUsers(c.Request.Context(), domain.LearnerUserFilter{
		Query:    c.Query("q"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleUpdateLearnerUser(c *gin.Context) {
	learnerID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || learnerID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid learner id"))
		return
	}

	var input domain.UpdateLearnerUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateLearnerUser(c.Request.Context(), uint(learnerID), input)
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
