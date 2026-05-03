package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

const (
	inviteCommissionStatusPending   = "pending"
	inviteCommissionStatusPaid      = "paid"
	inviteCommissionStatusCancelled = "cancelled"

	inviteWithdrawStatusPending   = "pending"
	inviteWithdrawStatusApproved  = "approved"
	inviteWithdrawStatusRejected  = "rejected"
	inviteWithdrawStatusPaid      = "paid"
	inviteWithdrawStatusCancelled = "cancelled"

	invitePaymentTypeWechat = "wechat"
	invitePaymentTypeAlipay = "alipay"
)

type inviteCommissionStats struct {
	CommissionRate       float64
	AvailableCents       int64
	WithdrawingCents     int64
	PaidCents            int64
	TotalCents           int64
}

func (s *Service) GetInvitePayoutProfile(ctx context.Context, learnerID uint) (domain.InvitePayoutProfile, error) {
	if learnerID == 0 {
		return domain.InvitePayoutProfile{}, errors.New("learner id is required")
	}

	var model storage.InvitePayoutProfile
	if err := s.db.WithContext(ctx).
		Where("learner_user_id = ?", learnerID).
		First(&model).Error; err != nil {
		if isRecordNotFound(err) {
			return domain.InvitePayoutProfile{}, nil
		}
		return domain.InvitePayoutProfile{}, err
	}
	return toInvitePayoutProfile(model), nil
}

func (s *Service) SaveInvitePayoutProfile(ctx context.Context, learnerID uint, input domain.SaveInvitePayoutProfileInput) (domain.InvitePayoutProfile, error) {
	if learnerID == 0 {
		return domain.InvitePayoutProfile{}, errors.New("learner id is required")
	}

	var model storage.InvitePayoutProfile
	err := s.db.WithContext(ctx).
		Where("learner_user_id = ?", learnerID).
		First(&model).Error
	if err != nil && !isRecordNotFound(err) {
		return domain.InvitePayoutProfile{}, err
	}
	if isRecordNotFound(err) {
		model = storage.InvitePayoutProfile{
			LearnerUserID: learnerID,
		}
	}

	model.RealName = strings.TrimSpace(input.RealName)
	model.WechatAccount = strings.TrimSpace(input.WechatAccount)
	model.WechatQRCode = strings.TrimSpace(input.WechatQRCode)
	model.AlipayAccount = strings.TrimSpace(input.AlipayAccount)
	model.AlipayQRCode = strings.TrimSpace(input.AlipayQRCode)

	if model.ID == 0 {
		if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
			return domain.InvitePayoutProfile{}, err
		}
	} else {
		if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
			return domain.InvitePayoutProfile{}, err
		}
	}

	return toInvitePayoutProfile(model), nil
}

func (s *Service) ListInviteCommissionRecords(ctx context.Context, learnerID uint, filter domain.InviteCommissionFilter) (domain.PagedInviteCommissionRecords, error) {
	if learnerID == 0 {
		return domain.PagedInviteCommissionRecords{}, errors.New("learner id is required")
	}

	page, pageSize := normalizePage(filter.Page, filter.PageSize, 10)
	query := s.db.WithContext(ctx).
		Model(&storage.InviteCommissionRecord{}).
		Where("inviter_learner_user_id = ?", learnerID)

	status, err := normalizeInviteCommissionFilterStatus(filter.Status)
	if err != nil {
		return domain.PagedInviteCommissionRecords{}, err
	}
	switch status {
	case "", "all":
	case "available":
		query = query.Where("status = ? AND withdraw_request_id IS NULL", inviteCommissionStatusPending)
	case "withdrawing":
		query = query.Where("status = ? AND withdraw_request_id IS NOT NULL", inviteCommissionStatusPending)
	default:
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedInviteCommissionRecords{}, err
	}

	var models []storage.InviteCommissionRecord
	if err := query.
		Order("COALESCE(order_paid_at, created_at) desc, id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&models).Error; err != nil {
		return domain.PagedInviteCommissionRecords{}, err
	}

	learnerMap, err := s.inviteLearnerMapByIDs(ctx, invitedLearnerIDs(models))
	if err != nil {
		return domain.PagedInviteCommissionRecords{}, err
	}

	items := make([]domain.InviteCommissionRecord, 0, len(models))
	for _, model := range models {
		invited := learnerMap[model.InvitedLearnerUserID]
		items = append(items, domain.InviteCommissionRecord{
			ID:                 model.ID,
			PaymentOrderID:     model.PaymentOrderID,
			PaymentOrderNo:     model.PaymentOrderNo,
			InvitedUserID:      model.InvitedLearnerUserID,
			InvitedUsername:    invited.Username,
			InvitedDisplayName: invited.DisplayName,
			OrderAmountCents:   model.OrderAmountCents,
			CommissionRate:     model.CommissionRate,
			CommissionCents:    model.CommissionCents,
			Status:             model.Status,
			WithdrawRequestID:  model.WithdrawRequestID,
			OrderPaidAt:        model.OrderPaidAt,
			PaidAt:             model.PaidAt,
			CreatedAt:          model.CreatedAt,
		})
	}

	return domain.PagedInviteCommissionRecords{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ListInviteWithdrawRequests(ctx context.Context, learnerID uint, filter domain.InviteWithdrawFilter) (domain.PagedInviteWithdrawRequests, error) {
	if learnerID == 0 {
		return domain.PagedInviteWithdrawRequests{}, errors.New("learner id is required")
	}

	page, pageSize := normalizePage(filter.Page, filter.PageSize, 10)
	query := s.db.WithContext(ctx).
		Model(&storage.InviteWithdrawRequest{}).
		Where("learner_user_id = ?", learnerID)

	status, err := normalizeInviteWithdrawStatus(filter.Status, true)
	if err != nil {
		return domain.PagedInviteWithdrawRequests{}, err
	}
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"payment_type LIKE ? OR account_name LIKE ? OR account_no LIKE ? OR admin_note LIKE ?",
			like,
			like,
			like,
			like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedInviteWithdrawRequests{}, err
	}

	var models []storage.InviteWithdrawRequest
	if err := query.
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&models).Error; err != nil {
		return domain.PagedInviteWithdrawRequests{}, err
	}

	items := make([]domain.InviteWithdrawRequest, 0, len(models))
	for _, model := range models {
		items = append(items, toInviteWithdrawRequest(model))
	}

	return domain.PagedInviteWithdrawRequests{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) CreateInviteWithdrawRequest(ctx context.Context, learnerID uint, input domain.CreateInviteWithdrawRequestInput) (domain.InviteWithdrawRequest, error) {
	if learnerID == 0 {
		return domain.InviteWithdrawRequest{}, errors.New("learner id is required")
	}
	if input.AmountCents <= 0 {
		return domain.InviteWithdrawRequest{}, errors.New("withdraw amount must be greater than 0")
	}

	paymentType, err := normalizeInvitePaymentType(input.PaymentType)
	if err != nil {
		return domain.InviteWithdrawRequest{}, err
	}

	var result storage.InviteWithdrawRequest
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var payout storage.InvitePayoutProfile
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("learner_user_id = ?", learnerID).
			First(&payout).Error; err != nil {
			if isRecordNotFound(err) {
				return errors.New("please save your payout profile before creating a withdraw request")
			}
			return err
		}

		accountName, accountNo, accountQRCode, err := resolveInvitePayoutSnapshot(payout, paymentType)
		if err != nil {
			return err
		}

		var commissions []storage.InviteCommissionRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("inviter_learner_user_id = ? AND status = ? AND withdraw_request_id IS NULL", learnerID, inviteCommissionStatusPending).
			Order("created_at asc, id asc").
			Find(&commissions).Error; err != nil {
			return err
		}
		if len(commissions) == 0 {
			return errors.New("there is no available invite commission to withdraw")
		}

		var totalAvailable int64
		for _, item := range commissions {
			totalAvailable += item.CommissionCents
		}
		if input.AmountCents > totalAvailable {
			return fmt.Errorf("withdraw amount exceeds available commission; max available is %d cents", totalAvailable)
		}

		selectedIDs := make([]uint, 0, len(commissions))
		var selectedSum int64
		for _, item := range commissions {
			if selectedSum >= input.AmountCents {
				break
			}
			selectedIDs = append(selectedIDs, item.ID)
			selectedSum += item.CommissionCents
			if selectedSum == input.AmountCents {
				break
			}
		}
		if selectedSum != input.AmountCents {
			return fmt.Errorf(
				"withdraw amount must match an exact commission bundle; next available bundle is %d cents, full available is %d cents",
				selectedSum,
				totalAvailable,
			)
		}

		result = storage.InviteWithdrawRequest{
			LearnerUserID: learnerID,
			AmountCents:   input.AmountCents,
			PaymentType:   paymentType,
			AccountName:   accountName,
			AccountNo:     accountNo,
			AccountQRCode: accountQRCode,
			Status:        inviteWithdrawStatusPending,
		}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}

		if len(selectedIDs) == 0 {
			return errors.New("there is no commission selected for this withdraw request")
		}
		if err := tx.Model(&storage.InviteCommissionRecord{}).
			Where("id IN ? AND inviter_learner_user_id = ? AND status = ? AND withdraw_request_id IS NULL", selectedIDs, learnerID, inviteCommissionStatusPending).
			Update("withdraw_request_id", result.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return domain.InviteWithdrawRequest{}, err
	}

	return toInviteWithdrawRequest(result), nil
}

func (s *Service) CancelInviteWithdrawRequest(ctx context.Context, learnerID, requestID uint) error {
	if learnerID == 0 {
		return errors.New("learner id is required")
	}
	if requestID == 0 {
		return errors.New("withdraw request id is required")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model storage.InviteWithdrawRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND learner_user_id = ?", requestID, learnerID).
			First(&model).Error; err != nil {
			if isRecordNotFound(err) {
				return errors.New("withdraw request does not exist")
			}
			return err
		}
		if model.Status != inviteWithdrawStatusPending {
			return errors.New("only pending withdraw requests can be cancelled")
		}

		if err := tx.Model(&storage.InviteCommissionRecord{}).
			Where("withdraw_request_id = ? AND status = ?", model.ID, inviteCommissionStatusPending).
			Update("withdraw_request_id", nil).Error; err != nil {
			return err
		}

		now := time.Now()
		updates := map[string]any{
			"status":       inviteWithdrawStatusCancelled,
			"processed_at": &now,
		}
		return tx.Model(&model).Updates(updates).Error
	})
}

func (s *Service) ListAdminInviteWithdrawRequests(ctx context.Context, filter domain.InviteWithdrawFilter) (domain.PagedAdminInviteWithdrawRequests, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).
		Model(&storage.InviteWithdrawRequest{})

	status, err := normalizeInviteWithdrawStatus(filter.Status, true)
	if err != nil {
		return domain.PagedAdminInviteWithdrawRequests{}, err
	}
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Joins("LEFT JOIN learner_users ON learner_users.id = invite_withdraw_requests.learner_user_id").
			Where(
				"learner_users.username LIKE ? OR learner_users.display_name LIKE ? OR invite_withdraw_requests.account_name LIKE ? OR invite_withdraw_requests.account_no LIKE ? OR invite_withdraw_requests.admin_note LIKE ?",
				like,
				like,
				like,
				like,
				like,
			)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedAdminInviteWithdrawRequests{}, err
	}

	var models []storage.InviteWithdrawRequest
	if err := query.
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&models).Error; err != nil {
		return domain.PagedAdminInviteWithdrawRequests{}, err
	}

	items, err := s.adminInviteWithdrawItems(ctx, models)
	if err != nil {
		return domain.PagedAdminInviteWithdrawRequests{}, err
	}

	return domain.PagedAdminInviteWithdrawRequests{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) GetAdminInviteWithdrawDetail(ctx context.Context, requestID uint) (domain.AdminInviteWithdrawDetail, error) {
	if requestID == 0 {
		return domain.AdminInviteWithdrawDetail{}, errors.New("withdraw request id is required")
	}

	var withdraw storage.InviteWithdrawRequest
	if err := s.db.WithContext(ctx).
		Where("id = ?", requestID).
		First(&withdraw).Error; err != nil {
		if isRecordNotFound(err) {
			return domain.AdminInviteWithdrawDetail{}, errors.New("withdraw request does not exist")
		}
		return domain.AdminInviteWithdrawDetail{}, err
	}

	items, err := s.adminInviteWithdrawItems(ctx, []storage.InviteWithdrawRequest{withdraw})
	if err != nil {
		return domain.AdminInviteWithdrawDetail{}, err
	}
	if len(items) == 0 {
		return domain.AdminInviteWithdrawDetail{}, errors.New("withdraw request does not exist")
	}

	var commissions []storage.InviteCommissionRecord
	if err := s.db.WithContext(ctx).
		Where("withdraw_request_id = ?", withdraw.ID).
		Order("created_at asc, id asc").
		Find(&commissions).Error; err != nil {
		return domain.AdminInviteWithdrawDetail{}, err
	}

	learnerMap, err := s.inviteLearnerMapByIDs(ctx, invitedLearnerIDs(commissions))
	if err != nil {
		return domain.AdminInviteWithdrawDetail{}, err
	}

	commissionItems := make([]domain.InviteCommissionRecord, 0, len(commissions))
	for _, model := range commissions {
		invited := learnerMap[model.InvitedLearnerUserID]
		commissionItems = append(commissionItems, domain.InviteCommissionRecord{
			ID:                 model.ID,
			PaymentOrderID:     model.PaymentOrderID,
			PaymentOrderNo:     model.PaymentOrderNo,
			InvitedUserID:      model.InvitedLearnerUserID,
			InvitedUsername:    invited.Username,
			InvitedDisplayName: invited.DisplayName,
			OrderAmountCents:   model.OrderAmountCents,
			CommissionRate:     model.CommissionRate,
			CommissionCents:    model.CommissionCents,
			Status:             model.Status,
			WithdrawRequestID:  model.WithdrawRequestID,
			OrderPaidAt:        model.OrderPaidAt,
			PaidAt:             model.PaidAt,
			CreatedAt:          model.CreatedAt,
		})
	}

	return domain.AdminInviteWithdrawDetail{
		Withdraw:    items[0],
		Commissions: commissionItems,
	}, nil
}

func (s *Service) ApproveInviteWithdrawRequest(ctx context.Context, requestID, adminID uint, input domain.ProcessInviteWithdrawInput) (domain.AdminInviteWithdrawItem, error) {
	return s.processInviteWithdrawRequest(ctx, requestID, adminID, input, inviteWithdrawStatusApproved)
}

func (s *Service) RejectInviteWithdrawRequest(ctx context.Context, requestID, adminID uint, input domain.ProcessInviteWithdrawInput) (domain.AdminInviteWithdrawItem, error) {
	return s.processInviteWithdrawRequest(ctx, requestID, adminID, input, inviteWithdrawStatusRejected)
}

func (s *Service) PayInviteWithdrawRequest(ctx context.Context, requestID, adminID uint, input domain.ProcessInviteWithdrawInput) (domain.AdminInviteWithdrawItem, error) {
	return s.processInviteWithdrawRequest(ctx, requestID, adminID, input, inviteWithdrawStatusPaid)
}

func (s *Service) createInviteCommissionForPayment(tx *gorm.DB, order storage.PaymentOrder) error {
	customerRef := strings.TrimSpace(order.CustomerRef)
	if order.ID == 0 || customerRef == "" {
		return nil
	}

	var learner storage.LearnerUser
	if err := tx.Where("username = ?", customerRef).First(&learner).Error; err != nil {
		if isRecordNotFound(err) {
			return nil
		}
		return err
	}
	if learner.InvitedByUserID == nil || *learner.InvitedByUserID == 0 || *learner.InvitedByUserID == learner.ID {
		return nil
	}

	commissionRate, err := inviteCommissionRateTx(tx)
	if err != nil {
		return err
	}
	if commissionRate <= 0 {
		return nil
	}

	commissionCents := int64(float64(order.AmountCents) * commissionRate / 100.0)
	if commissionCents <= 0 {
		return nil
	}

	model := storage.InviteCommissionRecord{
		InviterLearnerUserID: *learner.InvitedByUserID,
		PaymentOrderID:       order.ID,
		PaymentOrderNo:       order.OrderNo,
		InvitedLearnerUserID: learner.ID,
		OrderAmountCents:     int64(order.AmountCents),
		CommissionRate:       commissionRate,
		CommissionCents:      commissionCents,
		Status:               inviteCommissionStatusPending,
		OrderPaidAt:          order.PaidAt,
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "payment_order_id"}},
		DoNothing: true,
	}).Create(&model).Error
}

func (s *Service) getInviteCommissionStats(ctx context.Context, learnerID uint) (inviteCommissionStats, error) {
	stats := inviteCommissionStats{}
	if learnerID == 0 {
		return stats, errors.New("learner id is required")
	}

	rate, err := s.getInviteCommissionRate(ctx)
	if err != nil {
		return stats, err
	}
	stats.CommissionRate = rate

	var rows []struct {
		Status      string
		HasWithdraw int
		AmountCents int64
	}
	if err := s.db.WithContext(ctx).
		Model(&storage.InviteCommissionRecord{}).
		Select("status, CASE WHEN withdraw_request_id IS NULL THEN 0 ELSE 1 END AS has_withdraw, COALESCE(SUM(commission_cents), 0) AS amount_cents").
		Where("inviter_learner_user_id = ?", learnerID).
		Group("status, CASE WHEN withdraw_request_id IS NULL THEN 0 ELSE 1 END").
		Scan(&rows).Error; err != nil {
		return stats, err
	}

	for _, row := range rows {
		switch row.Status {
		case inviteCommissionStatusPaid:
			stats.PaidCents += row.AmountCents
		case inviteCommissionStatusPending:
			if row.HasWithdraw > 0 {
				stats.WithdrawingCents += row.AmountCents
			} else {
				stats.AvailableCents += row.AmountCents
			}
		}
	}
	stats.TotalCents = stats.AvailableCents + stats.WithdrawingCents + stats.PaidCents
	return stats, nil
}

func (s *Service) getInviteCommissionRate(ctx context.Context) (float64, error) {
	model, err := s.ensureSiteSetting(ctx)
	if err != nil {
		return 0, err
	}
	if model.InviteCommissionRate < 0 {
		return 0, nil
	}
	return model.InviteCommissionRate, nil
}

func inviteCommissionRateTx(tx *gorm.DB) (float64, error) {
	var model storage.SiteSetting
	err := tx.Order("id asc").First(&model).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
		model = defaultSiteSetting()
		if err := tx.Create(&model).Error; err != nil {
			return 0, err
		}
	}
	if model.InviteCommissionRate < 0 {
		return 0, nil
	}
	return model.InviteCommissionRate, nil
}

func (s *Service) processInviteWithdrawRequest(ctx context.Context, requestID, adminID uint, input domain.ProcessInviteWithdrawInput, action string) (domain.AdminInviteWithdrawItem, error) {
	if requestID == 0 {
		return domain.AdminInviteWithdrawItem{}, errors.New("withdraw request id is required")
	}
	if adminID == 0 {
		return domain.AdminInviteWithdrawItem{}, errors.New("admin id is required")
	}

	if _, err := normalizeInviteWithdrawStatus(action, false); err != nil {
		return domain.AdminInviteWithdrawItem{}, err
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model storage.InviteWithdrawRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", requestID).
			First(&model).Error; err != nil {
			if isRecordNotFound(err) {
				return errors.New("withdraw request does not exist")
			}
			return err
		}

		now := time.Now()
		note := strings.TrimSpace(input.AdminNote)

		switch action {
		case inviteWithdrawStatusApproved:
			if model.Status != inviteWithdrawStatusPending {
				return errors.New("only pending withdraw requests can be approved")
			}
		case inviteWithdrawStatusRejected:
			if model.Status != inviteWithdrawStatusPending && model.Status != inviteWithdrawStatusApproved {
				return errors.New("only pending or approved withdraw requests can be rejected")
			}
			if err := tx.Model(&storage.InviteCommissionRecord{}).
				Where("withdraw_request_id = ? AND status = ?", model.ID, inviteCommissionStatusPending).
				Update("withdraw_request_id", nil).Error; err != nil {
				return err
			}
		case inviteWithdrawStatusPaid:
			if model.Status != inviteWithdrawStatusApproved {
				return errors.New("only approved withdraw requests can be marked as paid")
			}
			if err := tx.Model(&storage.InviteCommissionRecord{}).
				Where("withdraw_request_id = ? AND status = ?", model.ID, inviteCommissionStatusPending).
				Updates(map[string]any{
					"status":  inviteCommissionStatusPaid,
					"paid_at": &now,
				}).Error; err != nil {
				return err
			}
		default:
			return errors.New("unsupported withdraw action")
		}

		updates := map[string]any{
			"status":                     action,
			"admin_note":                 note,
			"processed_by_admin_user_id": adminID,
			"processed_at":               &now,
		}
		return tx.Model(&model).Updates(updates).Error
	})
	if err != nil {
		return domain.AdminInviteWithdrawItem{}, err
	}

	return s.getAdminInviteWithdrawItem(ctx, requestID)
}

func (s *Service) getAdminInviteWithdrawItem(ctx context.Context, requestID uint) (domain.AdminInviteWithdrawItem, error) {
	var model storage.InviteWithdrawRequest
	if err := s.db.WithContext(ctx).
		Where("id = ?", requestID).
		First(&model).Error; err != nil {
		if isRecordNotFound(err) {
			return domain.AdminInviteWithdrawItem{}, errors.New("withdraw request does not exist")
		}
		return domain.AdminInviteWithdrawItem{}, err
	}

	items, err := s.adminInviteWithdrawItems(ctx, []storage.InviteWithdrawRequest{model})
	if err != nil {
		return domain.AdminInviteWithdrawItem{}, err
	}
	if len(items) == 0 {
		return domain.AdminInviteWithdrawItem{}, errors.New("withdraw request does not exist")
	}
	return items[0], nil
}

func (s *Service) adminInviteWithdrawItems(ctx context.Context, models []storage.InviteWithdrawRequest) ([]domain.AdminInviteWithdrawItem, error) {
	items := make([]domain.AdminInviteWithdrawItem, 0, len(models))
	if len(models) == 0 {
		return items, nil
	}

	learnerIDs := make([]uint, 0, len(models))
	adminIDs := make([]uint, 0, len(models))
	for _, model := range models {
		learnerIDs = append(learnerIDs, model.LearnerUserID)
		if model.ProcessedByAdminUserID != nil && *model.ProcessedByAdminUserID > 0 {
			adminIDs = append(adminIDs, *model.ProcessedByAdminUserID)
		}
	}

	learnerMap, err := s.inviteLearnerMapByIDs(ctx, learnerIDs)
	if err != nil {
		return nil, err
	}
	adminMap, err := s.inviteAdminMapByIDs(ctx, adminIDs)
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		learner := learnerMap[model.LearnerUserID]
		adminName := ""
		if model.ProcessedByAdminUserID != nil {
			adminName = adminMap[*model.ProcessedByAdminUserID].DisplayName
			if strings.TrimSpace(adminName) == "" {
				adminName = adminMap[*model.ProcessedByAdminUserID].Username
			}
		}
		items = append(items, domain.AdminInviteWithdrawItem{
			ID:                 model.ID,
			LearnerUserID:      model.LearnerUserID,
			LearnerUsername:    learner.Username,
			LearnerDisplayName: learner.DisplayName,
			AmountCents:        model.AmountCents,
			PaymentType:        model.PaymentType,
			AccountName:        model.AccountName,
			AccountNo:          model.AccountNo,
			AccountQRCode:      model.AccountQRCode,
			Status:             model.Status,
			AdminNote:          model.AdminNote,
			ProcessedByAdminID: model.ProcessedByAdminUserID,
			ProcessedByName:    adminName,
			ProcessedAt:        model.ProcessedAt,
			CreatedAt:          model.CreatedAt,
		})
	}

	return items, nil
}

func (s *Service) inviteLearnerMapByIDs(ctx context.Context, ids []uint) (map[uint]storage.LearnerUser, error) {
	result := make(map[uint]storage.LearnerUser)
	if len(ids) == 0 {
		return result, nil
	}

	uniqueIDs := uniqueUintSlice(ids)
	var learners []storage.LearnerUser
	if err := s.db.WithContext(ctx).
		Where("id IN ?", uniqueIDs).
		Find(&learners).Error; err != nil {
		return nil, err
	}
	for _, learner := range learners {
		result[learner.ID] = learner
	}
	return result, nil
}

func (s *Service) inviteAdminMapByIDs(ctx context.Context, ids []uint) (map[uint]storage.AdminUser, error) {
	result := make(map[uint]storage.AdminUser)
	if len(ids) == 0 {
		return result, nil
	}

	uniqueIDs := uniqueUintSlice(ids)
	var admins []storage.AdminUser
	if err := s.db.WithContext(ctx).
		Where("id IN ?", uniqueIDs).
		Find(&admins).Error; err != nil {
		return nil, err
	}
	for _, admin := range admins {
		result[admin.ID] = admin
	}
	return result, nil
}

func toInvitePayoutProfile(model storage.InvitePayoutProfile) domain.InvitePayoutProfile {
	return domain.InvitePayoutProfile{
		RealName:      model.RealName,
		WechatAccount: model.WechatAccount,
		WechatQRCode:  model.WechatQRCode,
		AlipayAccount: model.AlipayAccount,
		AlipayQRCode:  model.AlipayQRCode,
	}
}

func toInviteWithdrawRequest(model storage.InviteWithdrawRequest) domain.InviteWithdrawRequest {
	return domain.InviteWithdrawRequest{
		ID:            model.ID,
		AmountCents:   model.AmountCents,
		PaymentType:   model.PaymentType,
		AccountName:   model.AccountName,
		AccountNo:     model.AccountNo,
		AccountQRCode: model.AccountQRCode,
		Status:        model.Status,
		AdminNote:     model.AdminNote,
		ProcessedAt:   model.ProcessedAt,
		CreatedAt:     model.CreatedAt,
	}
}

func normalizeInviteCommissionFilterStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return "all", nil
	case "available", "withdrawing":
		return strings.ToLower(strings.TrimSpace(value)), nil
	case inviteCommissionStatusPending, inviteCommissionStatusPaid, inviteCommissionStatusCancelled:
		return strings.ToLower(strings.TrimSpace(value)), nil
	default:
		return "", errors.New("commission status must be one of: all, available, withdrawing, pending, paid, cancelled")
	}
}

func normalizeInviteWithdrawStatus(value string, allowAll bool) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "":
		if allowAll {
			return "all", nil
		}
		return "", errors.New("withdraw action is required")
	case "all":
		if allowAll {
			return "all", nil
		}
		return "", errors.New("invalid withdraw action")
	case inviteWithdrawStatusPending, inviteWithdrawStatusApproved, inviteWithdrawStatusRejected, inviteWithdrawStatusPaid, inviteWithdrawStatusCancelled:
		return value, nil
	default:
		return "", errors.New("withdraw status must be one of: all, pending, approved, rejected, paid, cancelled")
	}
}

func normalizeInvitePaymentType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case invitePaymentTypeWechat:
		return invitePaymentTypeWechat, nil
	case invitePaymentTypeAlipay:
		return invitePaymentTypeAlipay, nil
	default:
		return "", errors.New("payment type must be wechat or alipay")
	}
}

func resolveInvitePayoutSnapshot(profile storage.InvitePayoutProfile, paymentType string) (accountName string, accountNo string, accountQRCode string, err error) {
	accountName = strings.TrimSpace(profile.RealName)
	switch paymentType {
	case invitePaymentTypeWechat:
		accountNo = strings.TrimSpace(profile.WechatAccount)
		accountQRCode = strings.TrimSpace(profile.WechatQRCode)
		if accountNo == "" && accountQRCode == "" {
			return "", "", "", errors.New("please save a wechat payout account or QR code first")
		}
	case invitePaymentTypeAlipay:
		accountNo = strings.TrimSpace(profile.AlipayAccount)
		accountQRCode = strings.TrimSpace(profile.AlipayQRCode)
		if accountNo == "" && accountQRCode == "" {
			return "", "", "", errors.New("please save an alipay payout account or QR code first")
		}
	default:
		return "", "", "", errors.New("unsupported payment type")
	}
	return accountName, accountNo, accountQRCode, nil
}

func invitedLearnerIDs(models []storage.InviteCommissionRecord) []uint {
	ids := make([]uint, 0, len(models))
	for _, model := range models {
		if model.InvitedLearnerUserID == 0 {
			continue
		}
		ids = append(ids, model.InvitedLearnerUserID)
	}
	return ids
}

func uniqueUintSlice(items []uint) []uint {
	if len(items) == 0 {
		return items
	}
	result := make([]uint, 0, len(items))
	seen := make(map[uint]struct{}, len(items))
	for _, item := range items {
		if item == 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
