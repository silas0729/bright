package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func (s *Service) ListLearnerPaymentOrders(ctx context.Context, customerRef string, filter domain.PaymentOrderFilter) (domain.PagedPaymentOrders, error) {
	customerRef = strings.TrimSpace(customerRef)
	if customerRef == "" {
		return domain.PagedPaymentOrders{}, errors.New("customer ref is required")
	}
	filter.CustomerRef = customerRef
	return s.ListPaymentOrders(ctx, filter)
}

func (s *Service) ListLearnerMemberships(ctx context.Context, customerRef string, filter domain.SubscriptionFilter) (domain.PagedSubscriptions, error) {
	customerRef = strings.TrimSpace(customerRef)
	if customerRef == "" {
		return domain.PagedSubscriptions{}, errors.New("customer ref is required")
	}
	filter.CustomerRef = customerRef
	return s.ListMemberSubscriptions(ctx, filter)
}

func (s *Service) GetInviteSummary(ctx context.Context, learnerID uint) (domain.InviteSummary, error) {
	if learnerID == 0 {
		return domain.InviteSummary{}, errors.New("learner id is required")
	}

	var learner storage.LearnerUser
	if err := s.db.WithContext(ctx).Where("id = ?", learnerID).First(&learner).Error; err != nil {
		return domain.InviteSummary{}, err
	}
	if err := s.ensureInviteCode(ctx, &learner); err != nil {
		return domain.InviteSummary{}, err
	}

	var invitees []storage.LearnerUser
	if err := s.db.WithContext(ctx).
		Where("invited_by_user_id = ?", learner.ID).
		Order("created_at desc, id desc").
		Find(&invitees).Error; err != nil {
		return domain.InviteSummary{}, err
	}

	orderStats, err := s.inviteeOrderStats(ctx, invitees)
	if err != nil {
		return domain.InviteSummary{}, err
	}
	commissionStats, err := s.getInviteCommissionStats(ctx, learnerID)
	if err != nil {
		return domain.InviteSummary{}, err
	}

	items := make([]domain.InviteeItem, 0, len(invitees))
	var paidInviteCount int64
	var totalRechargeCents int64
	for _, invitee := range invitees {
		stat := orderStats[invitee.Username]
		if stat.Count > 0 {
			paidInviteCount++
			totalRechargeCents += stat.AmountCents
		}
		items = append(items, domain.InviteeItem{
			UserID:             invitee.ID,
			Username:           invitee.Username,
			DisplayName:        invitee.DisplayName,
			CreatedAt:          invitee.CreatedAt,
			PaidOrderCount:     stat.Count,
			TotalRechargeCents: stat.AmountCents,
			LastPaidAt:         stat.LastPaidAt,
		})
	}

	return domain.InviteSummary{
		InviteCode:                 learner.InviteCode,
		InvitedCount:               int64(len(invitees)),
		PaidInviteCount:            paidInviteCount,
		TotalRechargeCents:         totalRechargeCents,
		CommissionRate:             commissionStats.CommissionRate,
		CommissionAvailableCents:   commissionStats.AvailableCents,
		CommissionWithdrawingCents: commissionStats.WithdrawingCents,
		CommissionPaidCents:        commissionStats.PaidCents,
		CommissionTotalCents:       commissionStats.TotalCents,
		Items:                      items,
	}, nil
}

func (s *Service) ListAdminInviteStats(ctx context.Context, filter domain.AdminInviteStatFilter) (domain.PagedAdminInviteStats, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)

	var invitees []storage.LearnerUser
	if err := s.db.WithContext(ctx).
		Where("invited_by_user_id IS NOT NULL").
		Order("created_at desc, id desc").
		Find(&invitees).Error; err != nil {
		return domain.PagedAdminInviteStats{}, err
	}
	if len(invitees) == 0 {
		return domain.PagedAdminInviteStats{
			Items:    []domain.AdminInviteStatItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	inviterIDs := make(map[uint]struct{})
	for _, invitee := range invitees {
		if invitee.InvitedByUserID != nil && *invitee.InvitedByUserID > 0 {
			inviterIDs[*invitee.InvitedByUserID] = struct{}{}
		}
	}

	idList := make([]uint, 0, len(inviterIDs))
	for id := range inviterIDs {
		idList = append(idList, id)
	}

	var inviters []storage.LearnerUser
	if err := s.db.WithContext(ctx).
		Where("id IN ?", idList).
		Find(&inviters).Error; err != nil {
		return domain.PagedAdminInviteStats{}, err
	}
	inviterMap := make(map[uint]storage.LearnerUser, len(inviters))
	for _, inviter := range inviters {
		if err := s.ensureInviteCode(ctx, &inviter); err != nil {
			return domain.PagedAdminInviteStats{}, err
		}
		inviterMap[inviter.ID] = inviter
	}

	orderStats, err := s.inviteeOrderStats(ctx, invitees)
	if err != nil {
		return domain.PagedAdminInviteStats{}, err
	}

	type aggregate struct {
		item domain.AdminInviteStatItem
	}
	aggregates := make(map[uint]*aggregate, len(inviterMap))
	for _, invitee := range invitees {
		if invitee.InvitedByUserID == nil || *invitee.InvitedByUserID == 0 {
			continue
		}
		inviter, ok := inviterMap[*invitee.InvitedByUserID]
		if !ok {
			continue
		}
		current, exists := aggregates[inviter.ID]
		if !exists {
			current = &aggregate{
				item: domain.AdminInviteStatItem{
					InviterUserID:      inviter.ID,
					InviterUsername:    inviter.Username,
					InviterDisplayName: inviter.DisplayName,
					InviteCode:         inviter.InviteCode,
				},
			}
			aggregates[inviter.ID] = current
		}
		current.item.InvitedCount++
		if current.item.LastInviteAt == nil || invitee.CreatedAt.After(*current.item.LastInviteAt) {
			lastInviteAt := invitee.CreatedAt
			current.item.LastInviteAt = &lastInviteAt
		}

		stat := orderStats[invitee.Username]
		if stat.Count > 0 {
			current.item.PaidInviteCount++
			current.item.TotalRechargeCents += stat.AmountCents
			if stat.LastPaidAt != nil && (current.item.LastPaidAt == nil || stat.LastPaidAt.After(*current.item.LastPaidAt)) {
				current.item.LastPaidAt = stat.LastPaidAt
			}
		}
	}

	items := make([]domain.AdminInviteStatItem, 0, len(aggregates))
	queryText := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, current := range aggregates {
		if queryText != "" {
			joined := strings.ToLower(strings.Join([]string{
				current.item.InviterUsername,
				current.item.InviterDisplayName,
				current.item.InviteCode,
			}, " "))
			if !strings.Contains(joined, queryText) {
				continue
			}
		}
		items = append(items, current.item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TotalRechargeCents == items[j].TotalRechargeCents {
			return items[i].InvitedCount > items[j].InvitedCount
		}
		return items[i].TotalRechargeCents > items[j].TotalRechargeCents
	})

	total := int64(len(items))
	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	return domain.PagedAdminInviteStats{
		Items:    items[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

type learnerOrderStat struct {
	Count       int64
	AmountCents int64
	LastPaidAt  *time.Time
}

func (s *Service) inviteeOrderStats(ctx context.Context, invitees []storage.LearnerUser) (map[string]learnerOrderStat, error) {
	result := make(map[string]learnerOrderStat)
	if len(invitees) == 0 {
		return result, nil
	}

	usernames := make([]string, 0, len(invitees))
	seen := make(map[string]struct{}, len(invitees))
	for _, invitee := range invitees {
		if invitee.Username == "" {
			continue
		}
		if _, ok := seen[invitee.Username]; ok {
			continue
		}
		seen[invitee.Username] = struct{}{}
		usernames = append(usernames, invitee.Username)
	}
	if len(usernames) == 0 {
		return result, nil
	}

	var rows []struct {
		CustomerRef string
		Count       int64
		AmountCents int64
		LastPaidAt  *time.Time
	}
	if err := s.db.WithContext(ctx).
		Model(&storage.PaymentOrder{}).
		Select("customer_ref, COUNT(*) AS count, COALESCE(SUM(amount_cents), 0) AS amount_cents, MAX(paid_at) AS last_paid_at").
		Where("customer_ref IN ? AND status = ?", usernames, paymentStatusSuccess).
		Group("customer_ref").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.CustomerRef] = learnerOrderStat{
			Count:       row.Count,
			AmountCents: row.AmountCents,
			LastPaidAt:  row.LastPaidAt,
		}
	}
	return result, nil
}

func (s *Service) ensureInviteCode(ctx context.Context, learner *storage.LearnerUser) error {
	if learner == nil {
		return errors.New("learner is required")
	}
	if strings.TrimSpace(learner.InviteCode) != "" {
		return nil
	}

	base := normalizeInviteCode(learner.Username)
	if base == "" {
		base = fmt.Sprintf("user-%d", learner.ID)
	}

	candidate := base
	for suffix := 1; ; suffix++ {
		var count int64
		if err := s.db.WithContext(ctx).
			Model(&storage.LearnerUser{}).
			Where("invite_code = ? AND id <> ?", candidate, learner.ID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			break
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}

	if err := s.db.WithContext(ctx).
		Model(&storage.LearnerUser{}).
		Where("id = ?", learner.ID).
		Update("invite_code", candidate).Error; err != nil {
		return err
	}
	learner.InviteCode = candidate
	return nil
}

func normalizeInviteCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ".", "-", "_", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}
