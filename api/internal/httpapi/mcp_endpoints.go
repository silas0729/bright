package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
	"brights/api/internal/mcp"
)

func (s *Server) handleLearnerMCPEndpoints(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	items, err := s.service.ListLearnerMCPEndpoints(c.Request.Context(), claims.UserID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	for index := range items {
		items[index] = s.applyEndpointRuntimeState(items[index])
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleCreateLearnerMCPEndpoint(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var input domain.CreateMCPEndpointInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.CreateLearnerMCPEndpoint(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	if item.Enabled && s.endpointManager != nil {
		if refreshErr := s.endpointManager.RefreshLearnerEndpoint(c.Request.Context(), item.ID, claims.UserID, item.URL); refreshErr != nil {
			item.LastError = refreshErr.Error()
			item.ConnectionStatus = "error"
			item.IsConnected = false
		}
		item = s.applyEndpointRuntimeState(item)
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateLearnerMCPEndpoint(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	endpointID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || endpointID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid endpoint id"))
		return
	}

	var input domain.UpdateMCPEndpointInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateLearnerMCPEndpoint(c.Request.Context(), claims.UserID, uint(endpointID), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	if s.endpointManager != nil {
		if item.Enabled {
			if refreshErr := s.endpointManager.RefreshLearnerEndpoint(c.Request.Context(), item.ID, claims.UserID, item.URL); refreshErr != nil {
				item.LastError = refreshErr.Error()
				item.ConnectionStatus = "error"
				item.IsConnected = false
			}
			item = s.applyEndpointRuntimeState(item)
		} else {
			_ = s.endpointManager.DisconnectEndpoint(c.Request.Context(), item.ID, claims.UserID)
			item.ConnectionStatus = "disconnected"
			item.IsConnected = false
			item.LastError = ""
		}
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleDeleteLearnerMCPEndpoint(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	endpointID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || endpointID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid endpoint id"))
		return
	}

	if err := s.service.DeleteLearnerMCPEndpoint(c.Request.Context(), claims.UserID, uint(endpointID)); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if s.endpointManager != nil {
		_ = s.endpointManager.DisconnectEndpoint(c.Request.Context(), uint(endpointID), claims.UserID)
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleLearnerMCPEndpointStatus(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	endpointID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || endpointID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid endpoint id"))
		return
	}

	item, err := s.service.GetLearnerMCPEndpoint(c.Request.Context(), claims.UserID, uint(endpointID))
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, s.applyEndpointRuntimeState(item))
}

func (s *Server) handleLearnerMCPEndpointTools(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	endpointID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || endpointID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid endpoint id"))
		return
	}

	endpoint, tools, err := mcp.EndpointToolsForLearner(c.Request.Context(), s.mcpServer, claims.UserID, uint(endpointID), c.Query("subject"))
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoint_id":   endpoint.ID,
		"endpoint_name": endpoint.Name,
		"tool_count":    len(tools),
		"tools":         tools,
	})
}

func (s *Server) handleRefreshLearnerMCPConnections(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	if s.endpointManager == nil {
		writeError(c, http.StatusInternalServerError, domainError("endpoint manager is not available"))
		return
	}

	if err := s.endpointManager.RefreshLearnerEndpoints(c.Request.Context(), claims.UserID); err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}

	items, err := s.service.ListLearnerMCPEndpoints(c.Request.Context(), claims.UserID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}

	for index := range items {
		items[index] = s.applyEndpointRuntimeState(items[index])
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"endpoints": items,
	})
}

func (s *Server) applyEndpointRuntimeState(item domain.MCPEndpoint) domain.MCPEndpoint {
	if s.endpointManager == nil {
		return item
	}

	snapshot := s.endpointManager.Snapshot(item.ID)
	if strings.TrimSpace(snapshot.Status) == "" {
		return item
	}

	item.ConnectionStatus = snapshot.Status
	item.IsConnected = snapshot.IsConnected
	item.LastError = snapshot.LastError
	item.ConnectedAt = snapshot.ConnectedAt
	return item
}
