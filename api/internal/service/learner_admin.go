package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func (s *Service) ListLearnerUsers(ctx context.Context, filter domain.LearnerUserFilter) (domain.PagedLearnerUsers, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).Model(&storage.LearnerUser{})

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("username LIKE ? OR display_name LIKE ?", like, like)
	}
	if status := strings.TrimSpace(strings.ToLower(filter.Status)); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedLearnerUsers{}, err
	}

	var models []storage.LearnerUser
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedLearnerUsers{}, err
	}

	usernames := make([]string, 0, len(models))
	for _, model := range models {
		usernames = append(usernames, model.Username)
	}

	orderCounts, lastPaidAt, err := s.learnerOrderSummary(ctx, usernames)
	if err != nil {
		return domain.PagedLearnerUsers{}, err
	}
	subscriptions, err := s.learnerSubscriptionSummary(ctx, usernames)
	if err != nil {
		return domain.PagedLearnerUsers{}, err
	}

	items := make([]domain.LearnerUserAdminItem, 0, len(models))
	now := time.Now()
	for _, model := range models {
		subscription, hasSubscription := subscriptions[model.Username]
		item := domain.LearnerUserAdminItem{
			ID:              model.ID,
			Username:        model.Username,
			DisplayName:     model.DisplayName,
			Status:          model.Status,
			CreatedAt:       model.CreatedAt,
			PurchaseCount:   orderCounts[model.Username],
			HasMembership:   hasSubscription,
			LastOrderPaidAt: lastPaidAt[model.Username],
		}
		if hasSubscription {
			status := subscription.Status
			if status == subscriptionStatusActive && subscription.CurrentPeriodEnd != nil && subscription.CurrentPeriodEnd.Before(now) {
				status = subscriptionStatusExpired
			}
			item.MembershipStatus = status
			item.CurrentPlanKey = subscription.PlanKey
			item.CurrentPeriodEnd = subscription.CurrentPeriodEnd
			item.LastMembershipAt = timePtr(subscription.UpdatedAt)
		}
		items = append(items, item)
	}

	return domain.PagedLearnerUsers{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) UpdateLearnerUser(ctx context.Context, learnerID uint, input domain.UpdateLearnerUserInput) (domain.LearnerUser, error) {
	if learnerID == 0 {
		return domain.LearnerUser{}, errors.New("learner id is required")
	}

	var model storage.LearnerUser
	if err := s.db.WithContext(ctx).Where("id = ?", learnerID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.LearnerUser{}, errors.New("learner does not exist")
		}
		return domain.LearnerUser{}, err
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = model.DisplayName
	}

	status := model.Status
	if strings.TrimSpace(input.Status) != "" {
		nextStatus, err := normalizeLearnerStatus(input.Status)
		if err != nil {
			return domain.LearnerUser{}, err
		}
		status = nextStatus
	}

	if err := s.db.WithContext(ctx).Model(&model).Updates(map[string]any{
		"display_name": displayName,
		"status":       status,
	}).Error; err != nil {
		return domain.LearnerUser{}, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", learnerID).First(&model).Error; err != nil {
		return domain.LearnerUser{}, err
	}
	return toLearnerUser(model), nil
}

func (s *Service) learnerOrderSummary(ctx context.Context, usernames []string) (map[string]int64, map[string]*time.Time, error) {
	counts := make(map[string]int64, len(usernames))
	lastPaidAt := make(map[string]*time.Time, len(usernames))
	if len(usernames) == 0 {
		return counts, lastPaidAt, nil
	}

	type countRow struct {
		CustomerRef string
		Count       int64
	}
	var countRows []countRow
	if err := s.db.WithContext(ctx).
		Model(&storage.PaymentOrder{}).
		Select("customer_ref, COUNT(*) AS count").
		Where("customer_ref IN ? AND status = ?", usernames, paymentStatusSuccess).
		Group("customer_ref").
		Scan(&countRows).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range countRows {
		counts[row.CustomerRef] = row.Count
	}

	var orders []storage.PaymentOrder
	if err := s.db.WithContext(ctx).
		Where("customer_ref IN ? AND status = ?", usernames, paymentStatusSuccess).
		Order("id desc").
		Find(&orders).Error; err != nil {
		return nil, nil, err
	}
	for _, order := range orders {
		if _, ok := lastPaidAt[order.CustomerRef]; ok {
			continue
		}
		lastPaidAt[order.CustomerRef] = order.PaidAt
	}

	return counts, lastPaidAt, nil
}

func (s *Service) learnerSubscriptionSummary(ctx context.Context, usernames []string) (map[string]storage.MemberSubscription, error) {
	result := make(map[string]storage.MemberSubscription, len(usernames))
	if len(usernames) == 0 {
		return result, nil
	}

	var rows []storage.MemberSubscription
	if err := s.db.WithContext(ctx).
		Where("customer_ref IN ?", usernames).
		Order("customer_ref asc, id desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, ok := result[row.CustomerRef]; ok {
			continue
		}
		result[row.CustomerRef] = row
	}
	return result, nil
}

func normalizeLearnerStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "active":
		return "active", nil
	case "disabled":
		return "disabled", nil
	default:
		return "", errors.New("learner status must be active or disabled")
	}
}

func timePtr(value time.Time) *time.Time {
	v := value
	return &v
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
