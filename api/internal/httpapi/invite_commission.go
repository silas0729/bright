package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleLearnerInvitePayoutProfile(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	result, err := s.service.GetInvitePayoutProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerSaveInvitePayoutProfile(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var input domain.SaveInvitePayoutProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.SaveInvitePayoutProfile(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerInviteCommissions(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	result, err := s.service.ListInviteCommissionRecords(c.Request.Context(), claims.UserID, domain.InviteCommissionFilter{
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerInviteWithdraws(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	result, err := s.service.ListInviteWithdrawRequests(c.Request.Context(), claims.UserID, domain.InviteWithdrawFilter{
		Query:    c.Query("q"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerCreateInviteWithdraw(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var input domain.CreateInviteWithdrawRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.CreateInviteWithdrawRequest(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (s *Server) handleLearnerCancelInviteWithdraw(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || requestID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid withdraw request id"))
		return
	}

	if err := s.service.CancelInviteWithdrawRequest(c.Request.Context(), claims.UserID, uint(requestID)); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleAdminInviteWithdraws(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListAdminInviteWithdrawRequests(c.Request.Context(), domain.InviteWithdrawFilter{
		Query:    c.Query("q"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminInviteWithdrawDetail(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || requestID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid withdraw request id"))
		return
	}

	result, err := s.service.GetAdminInviteWithdrawDetail(c.Request.Context(), uint(requestID))
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminApproveInviteWithdraw(c *gin.Context) {
	s.handleAdminProcessInviteWithdraw(c, "approve")
}

func (s *Server) handleAdminRejectInviteWithdraw(c *gin.Context) {
	s.handleAdminProcessInviteWithdraw(c, "reject")
}

func (s *Server) handleAdminPayInviteWithdraw(c *gin.Context) {
	s.handleAdminProcessInviteWithdraw(c, "pay")
}

func (s *Server) handleAdminProcessInviteWithdraw(c *gin.Context, action string) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || requestID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid withdraw request id"))
		return
	}

	var input domain.ProcessInviteWithdrawInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	var result domain.AdminInviteWithdrawItem
	switch action {
	case "approve":
		result, err = s.service.ApproveInviteWithdrawRequest(c.Request.Context(), uint(requestID), claims.AdminID, input)
	case "reject":
		result, err = s.service.RejectInviteWithdrawRequest(c.Request.Context(), uint(requestID), claims.AdminID, input)
	case "pay":
		result, err = s.service.PayInviteWithdrawRequest(c.Request.Context(), uint(requestID), claims.AdminID, input)
	default:
		err = domainError("invalid withdraw action")
	}
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
