package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleLearnerPaymentOrders(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListLearnerPaymentOrders(c.Request.Context(), claims.Username, domain.PaymentOrderFilter{
		SubjectKey: c.Query("subject"),
		Status:     c.Query("status"),
		Query:      c.Query("q"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerMemberships(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListLearnerMemberships(c.Request.Context(), claims.Username, domain.SubscriptionFilter{
		SubjectKey: c.Query("subject"),
		Status:     c.Query("status"),
		Query:      c.Query("q"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerInviteSummary(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	result, err := s.service.GetInviteSummary(c.Request.Context(), claims.UserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminInviteStats(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListAdminInviteStats(c.Request.Context(), domain.AdminInviteStatFilter{
		Query:    c.Query("q"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
