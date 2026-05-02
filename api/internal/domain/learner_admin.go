package domain

import "time"

type LearnerUserFilter struct {
	Query    string
	Status   string
	Page     int
	PageSize int
}

type LearnerUserAdminItem struct {
	ID               uint       `json:"id"`
	Username         string     `json:"username"`
	DisplayName      string     `json:"display_name"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	PurchaseCount    int64      `json:"purchase_count"`
	HasMembership    bool       `json:"has_membership"`
	MembershipStatus string     `json:"membership_status"`
	CurrentPlanKey   string     `json:"current_plan_key"`
	CurrentPeriodEnd *time.Time `json:"current_period_end,omitempty"`
	LastOrderPaidAt  *time.Time `json:"last_order_paid_at,omitempty"`
	LastMembershipAt *time.Time `json:"last_membership_at,omitempty"`
}

type PagedLearnerUsers struct {
	Items    []LearnerUserAdminItem `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type UpdateLearnerUserInput struct {
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}
