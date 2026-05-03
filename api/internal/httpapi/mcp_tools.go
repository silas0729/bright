package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminMCPToolConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListAdminMCPToolConfigs(c.Request.Context(), domain.MCPToolConfigFilter{
		Query:    c.Query("q"),
		Category: c.Query("category"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
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
