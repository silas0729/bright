package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminMCPToolConfigs(c *gin.Context) {
	items, err := s.service.ListMCPToolConfigs(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
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
