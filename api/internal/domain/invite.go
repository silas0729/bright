package domain

import "time"

type InviteeItem struct {
	UserID             uint       `json:"user_id"`
	Username           string     `json:"username"`
	DisplayName        string     `json:"display_name"`
	CreatedAt          time.Time  `json:"created_at"`
	PaidOrderCount     int64      `json:"paid_order_count"`
	TotalRechargeCents int64      `json:"total_recharge_cents"`
	LastPaidAt         *time.Time `json:"last_paid_at,omitempty"`
}

type InviteSummary struct {
	InviteCode         string        `json:"invite_code"`
	InvitedCount       int64         `json:"invited_count"`
	PaidInviteCount    int64         `json:"paid_invite_count"`
	TotalRechargeCents int64         `json:"total_recharge_cents"`
	Items              []InviteeItem `json:"items"`
}

type AdminInviteStatItem struct {
	InviterUserID      uint       `json:"inviter_user_id"`
	InviterUsername    string     `json:"inviter_username"`
	InviterDisplayName string     `json:"inviter_display_name"`
	InviteCode         string     `json:"invite_code"`
	InvitedCount       int64      `json:"invited_count"`
	PaidInviteCount    int64      `json:"paid_invite_count"`
	TotalRechargeCents int64      `json:"total_recharge_cents"`
	LastInviteAt       *time.Time `json:"last_invite_at,omitempty"`
	LastPaidAt         *time.Time `json:"last_paid_at,omitempty"`
}

type PagedAdminInviteStats struct {
	Items    []AdminInviteStatItem `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type AdminInviteStatFilter struct {
	Query    string
	Page     int
	PageSize int
}
