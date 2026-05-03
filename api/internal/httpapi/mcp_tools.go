package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminMCPToolConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	isEnabled := parseOptionalBoolFilter(c.Query("enabled"))
	requiresMembership := parseOptionalBoolFilter(c.Query("requires_membership"))
	result, err := s.service.ListAdminMCPToolConfigs(c.Request.Context(), domain.MCPToolConfigFilter{
		Query:              c.Query("q"),
		Category:           c.Query("category"),
		IsEnabled:          isEnabled,
		RequiresMembership: requiresMembership,
		Page:               page,
		PageSize:           pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func parseOptionalBoolFilter(raw string) *bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "enabled", "active", "on", "true", "1", "yes":
		value := true
		return &value
	case "disabled", "inactive", "off", "false", "0", "no":
		value := false
		return &value
	default:
		return nil
	}
}

func (s *Server) handleAdminUpdateMCPToolConfig(c *gin.Context) {
	var input domain.UpdateMCPToolConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateMCPToolConfig(c.Request.Context(), c.Param("toolName"), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
