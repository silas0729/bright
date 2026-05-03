package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAccessibleAPIConfigMarket(c *gin.Context) {
	claims, hasClaims := s.optionalLearnerClaims(c)
	learnerID := uint(0)
	if hasClaims {
		learnerID = claims.UserID
	}

	items, err := s.service.ListAccessibleAPIConfigs(c.Request.Context(), learnerID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}

	queryText := strings.ToLower(strings.TrimSpace(c.Query("q")))
	category := strings.ToLower(strings.TrimSpace(c.Query("category")))
	filtered := make([]domain.APIConfig, 0, len(items))
	for _, item := range items {
		if category != "" && strings.ToLower(strings.TrimSpace(item.Category)) != category {
			continue
		}
		if queryText != "" {
			searchable := strings.ToLower(strings.Join([]string{
				item.Name,
				item.ResolvedToolName,
				item.Description,
				item.Category,
			}, " "))
			if !strings.Contains(searchable, queryText) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"items": filtered,
		"total": len(filtered),
	})
}

func (s *Server) handleLearnerAPIConfigs(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, err := s.service.ListLearnerAPIConfigs(c.Request.Context(), claims.UserID, domain.APIConfigFilter{
		Query:    c.Query("q"),
		Category: c.Query("category"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleCreateLearnerAPIConfig(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var input domain.CreateAPIConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.CreateLearnerAPIConfig(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateLearnerAPIConfig(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid api config id"))
		return
	}

	var input domain.UpdateAPIConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateLearnerAPIConfig(c.Request.Context(), claims.UserID, uint(id), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleDeleteLearnerAPIConfig(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid api config id"))
		return
	}

	if err := s.service.DeleteLearnerAPIConfig(c.Request.Context(), claims.UserID, uint(id)); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleTestLearnerAPIConfig(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid api config id"))
		return
	}

	var input domain.APIConfigTestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.TestLearnerAPIConfig(c.Request.Context(), claims.UserID, uint(id), input, domain.APIConfigExecutionContext{
		LearnerUserID: claims.UserID,
		Username:      claims.Username,
		SubjectKey:    firstNonEmpty(c.Query("subject"), c.Query("subject_key")),
		AccessToken:   bearerTokenFromRequest(c.Request),
		HTTPBaseURL:   httpRequestBaseURL(c.Request),
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminAPIConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, err := s.service.ListAdminAPIConfigs(c.Request.Context(), domain.APIConfigFilter{
		Query:      c.Query("q"),
		Category:   c.Query("category"),
		Page:       page,
		PageSize:   pageSize,
		IncludeAll: true,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleCreateAdminAPIConfig(c *gin.Context) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var input domain.CreateAPIConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.CreateAdminAPIConfig(c.Request.Context(), claims.AdminID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateAdminAPIConfig(c *gin.Context) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid api config id"))
		return
	}

	var input domain.UpdateAPIConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateAdminAPIConfig(c.Request.Context(), claims.AdminID, uint(id), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleDeleteAdminAPIConfig(c *gin.Context) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid api config id"))
		return
	}

	if err := s.service.DeleteAdminAPIConfig(c.Request.Context(), claims.AdminID, uint(id)); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleTestAdminAPIConfig(c *gin.Context) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid api config id"))
		return
	}

	var input domain.APIConfigTestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.TestAdminAPIConfig(c.Request.Context(), claims.AdminID, uint(id), input, domain.APIConfigExecutionContext{
		SubjectKey:  firstNonEmpty(c.Query("subject"), c.Query("subject_key")),
		AccessToken: bearerTokenFromRequest(c.Request),
		HTTPBaseURL: httpRequestBaseURL(c.Request),
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func bearerTokenFromRequest(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func httpRequestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		scheme = forwarded
	}
	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}
