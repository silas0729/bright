package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleLearnerLearningProgress(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListLearnerWordProgress(c.Request.Context(), claims.UserID, domain.LearnerWordProgressFilter{
		SubjectKey: c.Query("subject"),
		Query:      firstNonEmpty(c.Query("q"), c.Query("query")),
		Level:      c.Query("level"),
		Difficulty: c.Query("difficulty"),
		Page:       page,
		PageSize:   pageSize,
		DueOnly:    c.Query("due_only") == "true",
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerSaveLearningProgress(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var input domain.SaveLearnerWordProgressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.SaveLearnerWordProgress(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerReviewLearningWord(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var input domain.ReviewLearnerWordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.ReviewLearnerWord(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerLearningSummary(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	result, err := s.service.GetLearnerLearningSummary(c.Request.Context(), claims.UserID, c.Query("subject"))
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
