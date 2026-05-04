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
	CommissionRate     float64       `json:"commission_rate"`
	CommissionAvailableCents   int64 `json:"commission_available_cents"`
	CommissionWithdrawingCents int64 `json:"commission_withdrawing_cents"`
	CommissionPaidCents        int64 `json:"commission_paid_cents"`
	CommissionTotalCents       int64 `json:"commission_total_cents"`
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

type InvitePayoutProfile struct {
	RealName      string `json:"real_name"`
	WechatAccount string `json:"wechat_account"`
	WechatQRCode  string `json:"wechat_qr_code"`
	AlipayAccount string `json:"alipay_account"`
	AlipayQRCode  string `json:"alipay_qr_code"`
}

type SaveInvitePayoutProfileInput struct {
	RealName      string `json:"real_name"`
	WechatAccount string `json:"wechat_account"`
	WechatQRCode  string `json:"wechat_qr_code"`
	AlipayAccount string `json:"alipay_account"`
	AlipayQRCode  string `json:"alipay_qr_code"`
}

type InviteCommissionRecord struct {
	ID                 uint       `json:"id"`
	PaymentOrderID     uint       `json:"payment_order_id"`
	PaymentOrderNo     string     `json:"payment_order_no"`
	InvitedUserID      uint       `json:"invited_user_id"`
	InvitedUsername    string     `json:"invited_username"`
	InvitedDisplayName string     `json:"invited_display_name"`
	OrderAmountCents   int64      `json:"order_amount_cents"`
	CommissionRate     float64    `json:"commission_rate"`
	CommissionCents    int64      `json:"commission_cents"`
	Status             string     `json:"status"`
	WithdrawRequestID  *uint      `json:"withdraw_request_id,omitempty"`
	OrderPaidAt        *time.Time `json:"order_paid_at,omitempty"`
	PaidAt             *time.Time `json:"paid_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type InviteCommissionFilter struct {
	Status   string
	Page     int
	PageSize int
}

type PagedInviteCommissionRecords struct {
	Items    []InviteCommissionRecord `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

type InviteWithdrawRequest struct {
	ID            uint       `json:"id"`
	AmountCents   int64      `json:"amount_cents"`
	PaymentType   string     `json:"payment_type"`
	AccountName   string     `json:"account_name"`
	AccountNo     string     `json:"account_no"`
	AccountQRCode string     `json:"account_qr_code"`
	Status        string     `json:"status"`
	AdminNote     string     `json:"admin_note"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type CreateInviteWithdrawRequestInput struct {
	AmountCents int64  `json:"amount_cents"`
	PaymentType string `json:"payment_type"`
}

type InviteWithdrawFilter struct {
	Query    string
	Status   string
	Page     int
	PageSize int
}

type PagedInviteWithdrawRequests struct {
	Items    []InviteWithdrawRequest `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type AdminInviteWithdrawItem struct {
	ID                 uint       `json:"id"`
	LearnerUserID      uint       `json:"learner_user_id"`
	LearnerUsername    string     `json:"learner_username"`
	LearnerDisplayName string     `json:"learner_display_name"`
	AmountCents        int64      `json:"amount_cents"`
	PaymentType        string     `json:"payment_type"`
	AccountName        string     `json:"account_name"`
	AccountNo          string     `json:"account_no"`
	AccountQRCode      string     `json:"account_qr_code"`
	Status             string     `json:"status"`
	AdminNote          string     `json:"admin_note"`
	ProcessedByAdminID *uint      `json:"processed_by_admin_id,omitempty"`
	ProcessedByName    string     `json:"processed_by_name,omitempty"`
	ProcessedAt        *time.Time `json:"processed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type PagedAdminInviteWithdrawRequests struct {
	Items    []AdminInviteWithdrawItem `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

type AdminInviteWithdrawDetail struct {
	Withdraw    AdminInviteWithdrawItem  `json:"withdraw"`
	Commissions []InviteCommissionRecord `json:"commissions"`
}

type ProcessInviteWithdrawInput struct {
	AdminNote string `json:"admin_note"`
}
