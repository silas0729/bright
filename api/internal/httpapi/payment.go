package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminGetWechatPayConfig(c *gin.Context) {
	config, exists, err := s.service.GetWechatPayConfig(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	if !exists {
		c.JSON(http.StatusOK, gin.H{"exists": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exists": true,
		"config": config,
	})
}

func (s *Server) handleAdminSaveWechatPayConfig(c *gin.Context) {
	var input domain.SaveWechatPayConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	config, err := s.service.SaveWechatPayConfig(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (s *Server) handleAdminPaymentOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListPaymentOrders(c.Request.Context(), domain.PaymentOrderFilter{
		SubjectKey:  c.Query("subject"),
		PlanKey:     c.Query("plan_key"),
		CustomerRef: c.Query("customer_ref"),
		Status:      c.Query("status"),
		Query:       c.Query("q"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminSubscriptions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListMemberSubscriptions(c.Request.Context(), domain.SubscriptionFilter{
		SubjectKey:  c.Query("subject"),
		PlanKey:     c.Query("plan_key"),
		CustomerRef: c.Query("customer_ref"),
		Status:      c.Query("status"),
		Query:       c.Query("q"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminPaymentOrderDetail(c *gin.Context) {
	result, err := s.service.GetPaymentOrderStatus(c.Request.Context(), c.Param("orderNo"), "")
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			status = http.StatusNotFound
		}
		writeError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	item, err := s.service.GetMemberSubscriptionByID(c.Request.Context(), uint(id))
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

func (s *Server) handleAdminUpdateSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var input domain.UpdateSubscriptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateMemberSubscription(c.Request.Context(), uint(id), input)
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

func (s *Server) handleCreateWechatOrder(c *gin.Context) {
	var input domain.CreateWechatOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	order, err := s.service.CreateWechatNativeOrder(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, order)
}

func (s *Server) handleGetWechatOrderStatus(c *gin.Context) {
	result, err := s.service.GetPaymentOrderStatus(
		c.Request.Context(),
		c.Param("orderNo"),
		c.Query("customer_ref"),
	)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			status = http.StatusNotFound
		}
		writeError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleWechatPayNotify(c *gin.Context) {
	transaction, err := s.service.ParseWechatPayNotification(c.Request.Context(), c.Request)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "invalid notification") ||
			strings.Contains(strings.ToLower(err.Error()), "unsupported wechatpay-signature-type") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{
			"code":    "FAIL",
			"message": err.Error(),
		})
		return
	}

	if err := s.service.HandleWechatPayNotification(c.Request.Context(), transaction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "FAIL",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "success",
	})
}
