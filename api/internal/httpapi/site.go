package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleSiteSettings(c *gin.Context) {
	item, err := s.service.GetSiteSetting(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleAdminSiteSettings(c *gin.Context) {
	item, err := s.service.GetSiteSetting(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleAdminSaveSiteSettings(c *gin.Context) {
	var input domain.SaveSiteSettingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.SaveSiteSetting(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
