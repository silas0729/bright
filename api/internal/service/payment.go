package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
	payutil "brights/api/internal/wechatpay"
)

const (
	paymentProviderWechat = "wechat"
	paymentStatusPending  = "pending"
	paymentStatusSuccess  = "success"
	paymentStatusFailed   = "failed"
	paymentStatusClosed   = "closed"

	subscriptionStatusPending   = "pending"
	subscriptionStatusActive    = "active"
	subscriptionStatusExpired   = "expired"
	subscriptionStatusCancelled = "cancelled"
)

func (s *Service) GetWechatPayConfig(ctx context.Context) (domain.WechatPayConfig, bool, error) {
	model, exists, err := s.getWechatPayConfigModel(ctx)
	if err != nil || !exists {
		return domain.WechatPayConfig{}, exists, err
	}
	return toWechatPayConfig(model), true, nil
}

func (s *Service) ListPaymentOrders(ctx context.Context, filter domain.PaymentOrderFilter) (domain.PagedPaymentOrders, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).Model(&storage.PaymentOrder{})

	if subjectKey := normalizeKey(filter.SubjectKey); subjectKey != "" {
		query = query.Where("subject_key = ?", subjectKey)
	}
	if planKey := normalizeKey(filter.PlanKey); planKey != "" {
		query = query.Where("plan_key = ?", planKey)
	}
	if customerRef := strings.TrimSpace(filter.CustomerRef); customerRef != "" {
		query = query.Where("customer_ref = ?", customerRef)
	}
	if status := strings.TrimSpace(strings.ToLower(filter.Status)); status != "" {
		query = query.Where("status = ?", status)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"order_no LIKE ? OR customer_ref LIKE ? OR plan_key LIKE ? OR description LIKE ? OR provider_trade_no LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedPaymentOrders{}, err
	}

	var models []storage.PaymentOrder
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedPaymentOrders{}, err
	}

	items := make([]domain.WechatOrder, 0, len(models))
	for _, model := range models {
		items = append(items, toWechatOrder(model))
	}

	return domain.PagedPaymentOrders{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ListMemberSubscriptions(ctx context.Context, filter domain.SubscriptionFilter) (domain.PagedSubscriptions, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).Model(&storage.MemberSubscription{})

	if subjectKey := normalizeKey(filter.SubjectKey); subjectKey != "" {
		query = query.Where("subject_key = ?", subjectKey)
	}
	if planKey := normalizeKey(filter.PlanKey); planKey != "" {
		query = query.Where("plan_key = ?", planKey)
	}
	if customerRef := strings.TrimSpace(filter.CustomerRef); customerRef != "" {
		query = query.Where("customer_ref = ?", customerRef)
	}

	status := strings.TrimSpace(strings.ToLower(filter.Status))
	switch status {
	case "", "all":
	case subscriptionStatusExpired:
		now := time.Now()
		query = query.Where(
			"(status = ? AND current_period_end IS NOT NULL AND current_period_end < ?) OR status = ?",
			subscriptionStatusActive,
			now,
			subscriptionStatusExpired,
		)
	default:
		query = query.Where("status = ?", status)
		if status == subscriptionStatusActive {
			now := time.Now()
			query = query.Where("(current_period_end IS NULL OR current_period_end >= ?)", now)
		}
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"customer_ref LIKE ? OR plan_key LIKE ? OR subject_key LIKE ? OR provider_contract_id LIKE ?",
			like,
			like,
			like,
			like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedSubscriptions{}, err
	}

	var models []storage.MemberSubscription
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedSubscriptions{}, err
	}

	items := make([]domain.SubscriptionStatus, 0, len(models))
	for _, model := range models {
		item := toSubscriptionStatus(model)
		if item.Status == subscriptionStatusActive && item.CurrentPeriodEnd != nil && item.CurrentPeriodEnd.Before(time.Now()) {
			item.Status = subscriptionStatusExpired
		}
		items = append(items, item)
	}

	return domain.PagedSubscriptions{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) GetMemberSubscriptionByID(ctx context.Context, id uint) (domain.SubscriptionStatus, error) {
	if id == 0 {
		return domain.SubscriptionStatus{}, errors.New("member subscription id is required")
	}

	var model storage.MemberSubscription
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.SubscriptionStatus{}, errors.New("member subscription does not exist")
		}
		return domain.SubscriptionStatus{}, err
	}

	item := toSubscriptionStatus(model)
	if item.Status == subscriptionStatusActive && item.CurrentPeriodEnd != nil && item.CurrentPeriodEnd.Before(time.Now()) {
		item.Status = subscriptionStatusExpired
	}
	return item, nil
}

func (s *Service) UpdateMemberSubscription(ctx context.Context, id uint, input domain.UpdateSubscriptionInput) (domain.SubscriptionStatus, error) {
	if id == 0 {
		return domain.SubscriptionStatus{}, errors.New("member subscription id is required")
	}

	var model storage.MemberSubscription
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.SubscriptionStatus{}, errors.New("member subscription does not exist")
		}
		return domain.SubscriptionStatus{}, err
	}

	if planKey := normalizeKey(input.PlanKey); planKey != "" && planKey != model.PlanKey {
		plan, err := s.resolvePlan(ctx, 0, planKey)
		if err != nil {
			return domain.SubscriptionStatus{}, err
		}
		model.PlanID = uintPtr(plan.ID)
		model.PlanKey = plan.Key
	}

	status, err := normalizeSubscriptionStatus(input.Status)
	if err != nil {
		return domain.SubscriptionStatus{}, err
	}

	startedAt, err := parseOptionalTimestamp("started_at", input.StartedAt, input.ClearStartedAt)
	if err != nil {
		return domain.SubscriptionStatus{}, err
	}
	currentPeriodStart, err := parseOptionalTimestamp("current_period_start", input.CurrentPeriodStart, input.ClearCurrentPeriodStart)
	if err != nil {
		return domain.SubscriptionStatus{}, err
	}
	currentPeriodEnd, err := parseOptionalTimestamp("current_period_end", input.CurrentPeriodEnd, input.ClearCurrentPeriodEnd)
	if err != nil {
		return domain.SubscriptionStatus{}, err
	}
	cancelledAt, err := parseOptionalTimestamp("cancelled_at", input.CancelledAt, input.ClearCancelledAt)
	if err != nil {
		return domain.SubscriptionStatus{}, err
	}

	if currentPeriodStart != nil && currentPeriodEnd != nil && currentPeriodEnd.Before(*currentPeriodStart) {
		return domain.SubscriptionStatus{}, errors.New("current_period_end cannot be earlier than current_period_start")
	}

	model.Status = status
	model.AutoRenew = input.AutoRenew
	model.StartedAt = startedAt
	model.CurrentPeriodStart = currentPeriodStart
	model.CurrentPeriodEnd = currentPeriodEnd
	model.CancelledAt = cancelledAt

	now := time.Now()
	switch status {
	case subscriptionStatusCancelled:
		if model.CancelledAt == nil {
			model.CancelledAt = &now
		}
	case subscriptionStatusExpired:
		if model.CurrentPeriodEnd == nil {
			model.CurrentPeriodEnd = &now
		}
		model.AutoRenew = false
	case subscriptionStatusActive, subscriptionStatusPending:
		model.CancelledAt = nil
	}

	if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
		return domain.SubscriptionStatus{}, err
	}

	return s.GetMemberSubscriptionByID(ctx, id)
}

func (s *Service) SaveWechatPayConfig(ctx context.Context, input domain.SaveWechatPayConfigInput) (domain.WechatPayConfig, error) {
	model, exists, err := s.getWechatPayConfigModel(ctx)
	if err != nil {
		return domain.WechatPayConfig{}, err
	}
	if !exists {
		model = storage.WechatPayConfig{}
	}

	model.AuthMode = payutil.NormalizeAuthMode(input.AuthMode)
	model.MchID = strings.TrimSpace(input.MchID)
	model.AppID = strings.TrimSpace(input.AppID)
	model.MerchantSerialNo = strings.TrimSpace(input.MerchantSerialNo)
	model.PlatformCertSerialNo = strings.TrimSpace(input.PlatformCertSerialNo)
	model.NotifyURL = strings.TrimSpace(input.NotifyURL)
	model.DescriptionPrefix = payutil.NormalizeDescriptionPrefix(input.DescriptionPrefix)
	model.TimeExpireMinutes = payutil.NormalizeTimeExpireMinutes(input.TimeExpireMinutes)
	model.WechatPayPublicKeyID = strings.TrimSpace(input.WechatPayPublicKeyID)

	if model.AuthMode == "" {
		return domain.WechatPayConfig{}, errors.New("验签方式仅支持“微信支付公钥模式”或“平台证书自动下载模式”")
	}
	if model.MchID == "" {
		return domain.WechatPayConfig{}, errors.New("请先填写商户号")
	}
	if model.MerchantSerialNo == "" {
		return domain.WechatPayConfig{}, errors.New("请先填写商户证书序列号")
	}

	if input.ClearAPIv3Key {
		model.APIv3KeyEnc = ""
	} else if apiV3Key := strings.TrimSpace(input.APIv3Key); apiV3Key != "" {
		model.APIv3KeyEnc = payutil.EncryptConfigValue(apiV3Key, payutil.ConfigEncKey())
	}
	applyStoredConfigMaterial(&model.WechatPayPublicKeyPath, input.WechatPayPublicKey, input.ClearWechatPayPublicKey)
	applyStoredConfigMaterial(&model.CertPemPath, input.CertPem, input.ClearCertPem)
	applyStoredConfigMaterial(&model.KeyPemPath, input.KeyPem, input.ClearKeyPem)
	applyStoredConfigMaterial(&model.PlatformCertPath, input.PlatformCert, input.ClearPlatformCert)

	runtimeConfig, apiV3Key := buildWechatPayRuntimeConfig(model)
	if err := payutil.ValidateConfig(runtimeConfig, apiV3Key, false); err != nil {
		return domain.WechatPayConfig{}, err
	}

	if model.ID == 0 {
		if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
			return domain.WechatPayConfig{}, err
		}
	} else {
		if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
			return domain.WechatPayConfig{}, err
		}
	}

	return toWechatPayConfig(model), nil
}

func (s *Service) CreateWechatNativeOrder(ctx context.Context, input domain.CreateWechatOrderInput) (domain.WechatOrder, error) {
	plan, err := s.resolvePlan(ctx, input.PlanID, input.PlanKey)
	if err != nil {
		return domain.WechatOrder{}, err
	}
	if plan.PriceCents <= 0 {
		return domain.WechatOrder{}, errors.New("套餐金额必须大于 0")
	}
	if !planSupportsWechatNative(plan) {
		return domain.WechatOrder{}, errors.New("当前套餐暂不支持微信支付")
	}

	customerRef := strings.TrimSpace(input.CustomerRef)
	if customerRef == "" {
		return domain.WechatOrder{}, errors.New("请先填写学习账号")
	}

	subjectKey := normalizeKey(input.SubjectKey)
	if subjectKey == "" {
		subjectKey = "english"
	}
	subject, err := s.ensureSubject(ctx, subjectKey)
	if err != nil {
		return domain.WechatOrder{}, err
	}

	cfg, apiV3Key, err := s.loadWechatPayRuntimeConfig(ctx)
	if err != nil {
		return domain.WechatOrder{}, err
	}
	if err := payutil.ValidateConfig(cfg, apiV3Key, true); err != nil {
		return domain.WechatOrder{}, err
	}

	apiService, err := payutil.BuildNativeService(ctx, cfg, apiV3Key)
	if err != nil {
		return domain.WechatOrder{}, err
	}

	orderNo := generatePaymentOrderNo()
	description := buildPaymentDescription(subject.Name, plan.Name, input.Description, cfg.DescriptionPrefix)
	expiresAt := time.Now().Add(time.Duration(payutil.NormalizeTimeExpireMinutes(cfg.TimeExpireMinutes)) * time.Minute)

	model := storage.PaymentOrder{
		OrderNo:      orderNo,
		PlanID:       uintPtr(plan.ID),
		PlanKey:      plan.Key,
		SubjectKey:   subjectKey,
		CustomerRef:  customerRef,
		Description:  description,
		BillingMode:  plan.BillingMode,
		AmountCents:  plan.PriceCents,
		Currency:     "CNY",
		Provider:     paymentProviderWechat,
		Status:       paymentStatusPending,
		ExpiresAt:    &expiresAt,
		ErrorMessage: "",
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.WechatOrder{}, err
	}

	response, _, err := apiService.Prepay(ctx, native.PrepayRequest{
		Appid:       stringPtr(cfg.AppID),
		Mchid:       stringPtr(cfg.MchID),
		Description: stringPtr(description),
		OutTradeNo:  stringPtr(orderNo),
		TimeExpire:  &expiresAt,
		NotifyUrl:   stringPtr(cfg.NotifyURL),
		Amount: &native.Amount{
			Total:    int64Ptr(int64(plan.PriceCents)),
			Currency: stringPtr("CNY"),
		},
	})
	if err != nil {
		friendlyErr := payutil.NormalizeAPIError(err)
		updateErr := s.db.WithContext(ctx).Model(&storage.PaymentOrder{}).
			Where("id = ?", model.ID).
			Updates(map[string]any{
				"status":        paymentStatusFailed,
				"error_message": friendlyErr.Error(),
			}).Error
		if updateErr == nil {
			model.Status = paymentStatusFailed
			model.ErrorMessage = friendlyErr.Error()
		}
		return domain.WechatOrder{}, fmt.Errorf("创建微信支付订单失败：%w", friendlyErr)
	}

	if response != nil && response.CodeUrl != nil {
		model.CodeURL = strings.TrimSpace(*response.CodeUrl)
	}
	model.ErrorMessage = ""
	if err := s.db.WithContext(ctx).Model(&storage.PaymentOrder{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"code_url":      model.CodeURL,
			"error_message": "",
		}).Error; err != nil {
		return domain.WechatOrder{}, err
	}

	return toWechatOrder(model), nil
}

func (s *Service) GetPaymentOrderStatus(ctx context.Context, orderNo, customerRef string) (domain.PaymentOrderStatus, error) {
	orderNo = strings.TrimSpace(orderNo)
	customerRef = strings.TrimSpace(customerRef)
	if orderNo == "" {
		return domain.PaymentOrderStatus{}, errors.New("order_no is required")
	}

	var model storage.PaymentOrder
	if err := s.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.PaymentOrderStatus{}, errors.New("payment order does not exist")
		}
		return domain.PaymentOrderStatus{}, err
	}
	if customerRef != "" && customerRef != model.CustomerRef {
		return domain.PaymentOrderStatus{}, errors.New("payment order does not exist")
	}

	if model.Status == paymentStatusPending {
		_ = s.syncWechatOrderStatus(ctx, model.OrderNo)
		if err := s.db.WithContext(ctx).Where("id = ?", model.ID).First(&model).Error; err != nil {
			return domain.PaymentOrderStatus{}, err
		}
	}

	subscription, err := s.findLatestSubscriptionStatus(ctx, model.CustomerRef, model.SubjectKey)
	if err != nil {
		return domain.PaymentOrderStatus{}, err
	}

	return domain.PaymentOrderStatus{
		Order:        toWechatOrder(model),
		Subscription: subscription,
	}, nil
}

func (s *Service) HandleWechatPayNotification(ctx context.Context, transaction *payments.Transaction) error {
	if transaction == nil || transaction.OutTradeNo == nil {
		return errors.New("微信支付回调内容不完整")
	}
	return s.applyWechatTransaction(ctx, strings.TrimSpace(*transaction.OutTradeNo), transaction)
}

func (s *Service) ParseWechatPayNotification(ctx context.Context, request *http.Request) (*payments.Transaction, error) {
	cfg, apiV3Key, err := s.loadWechatPayRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}

	handler, err := payutil.BuildNotifyHandler(ctx, cfg, apiV3Key)
	if err != nil {
		return nil, err
	}

	transaction := new(payments.Transaction)
	if _, err := handler.ParseNotifyRequest(ctx, request, transaction); err != nil {
		return nil, err
	}
	return transaction, nil
}

func (s *Service) syncWechatOrderStatus(ctx context.Context, orderNo string) error {
	cfg, apiV3Key, err := s.loadWechatPayRuntimeConfig(ctx)
	if err != nil {
		return err
	}

	apiService, err := payutil.BuildNativeService(ctx, cfg, apiV3Key)
	if err != nil {
		return err
	}

	transaction, _, err := apiService.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: stringPtr(orderNo),
		Mchid:      stringPtr(cfg.MchID),
	})
	if err != nil {
		return payutil.NormalizeAPIError(err)
	}

	return s.applyWechatTransaction(ctx, orderNo, transaction)
}

func (s *Service) applyWechatTransaction(ctx context.Context, orderNo string, transaction *payments.Transaction) error {
	if transaction == nil {
		return nil
	}

	switch strings.ToUpper(strings.TrimSpace(stringValue(transaction.TradeState))) {
	case "SUCCESS":
		return s.markPaymentOrderSuccess(ctx, orderNo, transaction)
	case "CLOSED", "REVOKED":
		return s.updatePaymentOrderTerminalStatus(ctx, orderNo, paymentStatusClosed, transaction)
	case "PAYERROR":
		return s.updatePaymentOrderTerminalStatus(ctx, orderNo, paymentStatusFailed, transaction)
	default:
		return nil
	}
}

func (s *Service) markPaymentOrderSuccess(ctx context.Context, orderNo string, transaction *payments.Transaction) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order storage.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", orderNo).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("payment order does not exist")
			}
			return err
		}

		if order.Status == paymentStatusSuccess {
			return nil
		}

		if transaction != nil && transaction.Amount != nil && transaction.Amount.Total != nil {
			if int64(order.AmountCents) != *transaction.Amount.Total {
				return fmt.Errorf("payment amount mismatch for order %s", order.OrderNo)
			}
		}

		now := time.Now()
		order.Status = paymentStatusSuccess
		order.PaidAt = &now
		order.ErrorMessage = ""
		if transaction != nil && transaction.TransactionId != nil {
			order.ProviderTradeNo = strings.TrimSpace(*transaction.TransactionId)
		}

		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		return s.applyPaymentEntitlement(tx, order)
	})
}

func (s *Service) updatePaymentOrderTerminalStatus(ctx context.Context, orderNo, status string, transaction *payments.Transaction) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order storage.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", orderNo).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("payment order does not exist")
			}
			return err
		}

		if order.Status == paymentStatusSuccess {
			return nil
		}

		order.Status = status
		if transaction != nil && transaction.TransactionId != nil {
			order.ProviderTradeNo = strings.TrimSpace(*transaction.TransactionId)
		}
		if transaction != nil && transaction.TradeStateDesc != nil {
			order.ErrorMessage = strings.TrimSpace(*transaction.TradeStateDesc)
		}
		return tx.Save(&order).Error
	})
}

func (s *Service) applyPaymentEntitlement(tx *gorm.DB, order storage.PaymentOrder) error {
	plan, err := resolvePlanByOrder(tx, order)
	if err != nil {
		return err
	}

	var subscription storage.MemberSubscription
	query := tx.Where("customer_ref = ? AND subject_key = ?", order.CustomerRef, order.SubjectKey).
		Order("id desc")
	err = query.First(&subscription).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		subscription = storage.MemberSubscription{
			CustomerRef: order.CustomerRef,
			SubjectKey:  order.SubjectKey,
			Provider:    paymentProviderWechat,
		}
	}

	subscription.PlanID = order.PlanID
	subscription.PlanKey = order.PlanKey
	subscription.SubjectKey = order.SubjectKey
	subscription.Status = subscriptionStatusActive
	subscription.AutoRenew = false
	subscription.Provider = paymentProviderWechat
	subscription.CancelledAt = nil
	if subscription.StartedAt == nil {
		subscription.StartedAt = &now
	}

	switch strings.ToLower(strings.TrimSpace(plan.BillingMode)) {
	case "monthly":
		if subscription.ID != 0 && subscription.CurrentPeriodEnd == nil && subscription.Status == subscriptionStatusActive {
			if subscription.CurrentPeriodStart == nil {
				subscription.CurrentPeriodStart = &now
			}
			break
		}
		if subscription.CurrentPeriodEnd == nil || !subscription.CurrentPeriodEnd.After(now) {
			periodStart := now
			periodEnd := periodStart.AddDate(0, 1, 0)
			subscription.CurrentPeriodStart = &periodStart
			subscription.CurrentPeriodEnd = &periodEnd
		} else {
			periodStart := *subscription.CurrentPeriodEnd
			periodEnd := periodStart.AddDate(0, 1, 0)
			subscription.CurrentPeriodStart = &periodStart
			subscription.CurrentPeriodEnd = &periodEnd
		}
	default:
		if subscription.CurrentPeriodStart == nil {
			subscription.CurrentPeriodStart = &now
		}
		subscription.CurrentPeriodEnd = nil
	}

	if subscription.ID == 0 {
		return tx.Create(&subscription).Error
	}
	return tx.Save(&subscription).Error
}

func (s *Service) findLatestSubscriptionStatus(ctx context.Context, customerRef, subjectKey string) (*domain.SubscriptionStatus, error) {
	var model storage.MemberSubscription
	err := s.db.WithContext(ctx).
		Where("customer_ref = ? AND subject_key = ?", customerRef, subjectKey).
		Order("id desc").
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	status := toSubscriptionStatus(model)
	if status.Status == subscriptionStatusActive && status.CurrentPeriodEnd != nil && status.CurrentPeriodEnd.Before(time.Now()) {
		status.Status = subscriptionStatusExpired
	}
	return &status, nil
}

func (s *Service) LearnerHasActiveMembership(ctx context.Context, customerRef, subjectKey string) (bool, error) {
	customerRef = strings.TrimSpace(customerRef)
	subjectKey = normalizeKey(subjectKey)
	if customerRef == "" || subjectKey == "" {
		return false, nil
	}

	subscription, err := s.findLatestSubscriptionStatus(ctx, customerRef, subjectKey)
	if err != nil {
		return false, err
	}
	if subscription == nil || subscription.Status != subscriptionStatusActive {
		return false, nil
	}
	if subscription.CurrentPeriodEnd != nil && subscription.CurrentPeriodEnd.Before(time.Now()) {
		return false, nil
	}
	return true, nil
}

func (s *Service) getWechatPayConfigModel(ctx context.Context) (storage.WechatPayConfig, bool, error) {
	var model storage.WechatPayConfig
	err := s.db.WithContext(ctx).Order("id desc").First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return storage.WechatPayConfig{}, false, nil
		}
		return storage.WechatPayConfig{}, false, err
	}
	return model, true, nil
}

func (s *Service) loadWechatPayRuntimeConfig(ctx context.Context) (payutil.Config, string, error) {
	model, exists, err := s.getWechatPayConfigModel(ctx)
	if err != nil {
		return payutil.Config{}, "", err
	}
	if !exists {
		return payutil.Config{}, "", errors.New("还没有保存微信收款信息")
	}

	cfg, apiV3Key := buildWechatPayRuntimeConfig(model)
	return cfg, apiV3Key, nil
}

func (s *Service) resolvePlan(ctx context.Context, planID uint, planKey string) (storage.Plan, error) {
	var model storage.Plan
	switch {
	case planID > 0:
		if err := s.db.WithContext(ctx).Where("id = ?", planID).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.Plan{}, errors.New("plan does not exist")
			}
			return storage.Plan{}, err
		}
	case strings.TrimSpace(planKey) != "":
		if err := s.db.WithContext(ctx).Where("plan_key = ?", normalizeKey(planKey)).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.Plan{}, errors.New("plan does not exist")
			}
			return storage.Plan{}, err
		}
	default:
		return storage.Plan{}, errors.New("plan_id or plan_key is required")
	}
	return model, nil
}

func resolvePlanByOrder(tx *gorm.DB, order storage.PaymentOrder) (storage.Plan, error) {
	var plan storage.Plan
	switch {
	case order.PlanID != nil && *order.PlanID > 0:
		if err := tx.Where("id = ?", *order.PlanID).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.Plan{}, errors.New("plan does not exist")
			}
			return storage.Plan{}, err
		}
	case strings.TrimSpace(order.PlanKey) != "":
		if err := tx.Where("plan_key = ?", order.PlanKey).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.Plan{}, errors.New("plan does not exist")
			}
			return storage.Plan{}, err
		}
	default:
		return storage.Plan{}, errors.New("payment order plan info is missing")
	}
	return plan, nil
}

func toWechatPayConfig(model storage.WechatPayConfig) domain.WechatPayConfig {
	runtimeConfig, apiV3Key := buildWechatPayRuntimeConfig(model)
	readyForCheckout, validationError := payutil.CheckoutState(runtimeConfig, apiV3Key)

	return domain.WechatPayConfig{
		ID:                    model.ID,
		AuthMode:              runtimeConfig.AuthMode,
		MchID:                 model.MchID,
		AppID:                 model.AppID,
		MerchantSerialNo:      model.MerchantSerialNo,
		NotifyURL:             model.NotifyURL,
		DescriptionPrefix:     runtimeConfig.DescriptionPrefix,
		TimeExpireMinutes:     runtimeConfig.TimeExpireMinutes,
		WechatPayPublicKeyID:  model.WechatPayPublicKeyID,
		APIv3Key:              apiV3Key,
		WechatPayPublicKey:    runtimeConfig.WechatPayPublicKey,
		KeyPem:                runtimeConfig.KeyPem,
		HasAPIv3Key:           strings.TrimSpace(model.APIv3KeyEnc) != "",
		HasWechatPayPublicKey: hasStoredConfigMaterial(model.WechatPayPublicKeyPath),
		HasCertPem:            hasStoredConfigMaterial(model.CertPemPath),
		HasKeyPem:             hasStoredConfigMaterial(model.KeyPemPath),
		HasPlatformCert:       hasStoredConfigMaterial(model.PlatformCertPath),
		ReadyForCheckout:      readyForCheckout,
		ValidationError:       validationError,
		UpdatedAt:             model.UpdatedAt,
	}
}

func toWechatOrder(model storage.PaymentOrder) domain.WechatOrder {
	return domain.WechatOrder{
		OrderNo:         model.OrderNo,
		PlanID:          model.PlanID,
		PlanKey:         model.PlanKey,
		SubjectKey:      model.SubjectKey,
		CustomerRef:     model.CustomerRef,
		Description:     model.Description,
		BillingMode:     model.BillingMode,
		AmountCents:     model.AmountCents,
		Currency:        model.Currency,
		Provider:        model.Provider,
		ProviderTradeNo: model.ProviderTradeNo,
		CodeURL:         model.CodeURL,
		Status:          model.Status,
		ErrorMessage:    model.ErrorMessage,
		PaidAt:          model.PaidAt,
		ExpiresAt:       model.ExpiresAt,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func toSubscriptionStatus(model storage.MemberSubscription) domain.SubscriptionStatus {
	return domain.SubscriptionStatus{
		ID:                 model.ID,
		CustomerRef:        model.CustomerRef,
		PlanID:             model.PlanID,
		PlanKey:            model.PlanKey,
		SubjectKey:         model.SubjectKey,
		Status:             model.Status,
		AutoRenew:          model.AutoRenew,
		Provider:           model.Provider,
		ProviderContractID: model.ProviderContractID,
		StartedAt:          model.StartedAt,
		CurrentPeriodStart: model.CurrentPeriodStart,
		CurrentPeriodEnd:   model.CurrentPeriodEnd,
		CancelledAt:        model.CancelledAt,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func normalizeSubscriptionStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", subscriptionStatusActive:
		return subscriptionStatusActive, nil
	case subscriptionStatusPending:
		return subscriptionStatusPending, nil
	case subscriptionStatusExpired:
		return subscriptionStatusExpired, nil
	case subscriptionStatusCancelled:
		return subscriptionStatusCancelled, nil
	default:
		return "", errors.New("subscription status must be one of: pending, active, expired, cancelled")
	}
}

func parseOptionalTimestamp(fieldName, value string, clear bool) (*time.Time, error) {
	if clear {
		return nil, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("%s must be a valid datetime", fieldName)
}

func applyStoredConfigMaterial(target *string, plain string, clear bool) {
	if target == nil {
		return
	}
	if clear {
		*target = ""
		return
	}

	plain = strings.TrimSpace(plain)
	if plain == "" {
		return
	}
	*target = payutil.EncryptConfigValue(plain, payutil.ConfigEncKey())
}

func resolveStoredConfigMaterial(stored string) (content string, path string) {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return "", ""
	}

	if payutil.IsEncryptedConfigValue(stored) {
		return strings.TrimSpace(payutil.DecryptConfigValue(stored, payutil.ConfigEncKey())), ""
	}
	if strings.Contains(stored, "-----BEGIN") {
		return stored, ""
	}
	return "", stored
}

func hasStoredConfigMaterial(stored string) bool {
	return strings.TrimSpace(stored) != ""
}

func buildWechatPayRuntimeConfig(model storage.WechatPayConfig) (payutil.Config, string) {
	apiV3Key := payutil.DecryptConfigValue(model.APIv3KeyEnc, payutil.ConfigEncKey())

	cfg := payutil.Config{
		AuthMode:             payutil.NormalizeAuthMode(model.AuthMode),
		MchID:                model.MchID,
		AppID:                model.AppID,
		MerchantSerialNo:     model.MerchantSerialNo,
		APIv3KeyEnc:          model.APIv3KeyEnc,
		PlatformCertSerialNo: model.PlatformCertSerialNo,
		NotifyURL:            model.NotifyURL,
		DescriptionPrefix:    payutil.NormalizeDescriptionPrefix(model.DescriptionPrefix),
		TimeExpireMinutes:    payutil.NormalizeTimeExpireMinutes(model.TimeExpireMinutes),
		WechatPayPublicKeyID: model.WechatPayPublicKeyID,
		P12Path:              model.P12Path,
	}
	cfg.WechatPayPublicKey, cfg.WechatPayPublicKeyPath = resolveStoredConfigMaterial(model.WechatPayPublicKeyPath)
	cfg.CertPem, cfg.CertPemPath = resolveStoredConfigMaterial(model.CertPemPath)
	cfg.KeyPem, cfg.KeyPemPath = resolveStoredConfigMaterial(model.KeyPemPath)
	cfg.PlatformCert, cfg.PlatformCertPath = resolveStoredConfigMaterial(model.PlatformCertPath)

	return cfg, apiV3Key
}

func buildPaymentDescription(subjectName, planName, custom, prefix string) string {
	custom = strings.TrimSpace(custom)
	if custom != "" {
		return truncateString(custom, 120)
	}

	parts := make([]string, 0, 3)
	prefix = strings.TrimSpace(prefix)
	if prefix != "" {
		parts = append(parts, prefix)
	}
	if subjectName = strings.TrimSpace(subjectName); subjectName != "" {
		parts = append(parts, subjectName)
	}
	parts = append(parts, strings.TrimSpace(planName))
	return truncateString(strings.Join(parts, " "), 120)
}

func planSupportsWechatNative(plan storage.Plan) bool {
	if len(plan.PaymentChannels) == 0 {
		return true
	}
	for _, channel := range plan.PaymentChannels {
		switch strings.ToLower(strings.TrimSpace(channel)) {
		case "wechat", "wechat_native":
			return true
		}
	}
	return false
}

func generatePaymentOrderNo() string {
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	return fmt.Sprintf("BRI%s%x", time.Now().Format("20060102150405"), suffix)
}

func truncateString(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func stringPtr(value string) *string {
	v := value
	return &v
}

func int64Ptr(value int64) *int64 {
	v := value
	return &v
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
