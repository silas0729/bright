package domain

import "time"

type WechatPayConfig struct {
	ID                    uint      `json:"id"`
	AuthMode              string    `json:"auth_mode"`
	MchID                 string    `json:"mch_id"`
	AppID                 string    `json:"app_id"`
	MerchantSerialNo      string    `json:"merchant_serial_no"`
	NotifyURL             string    `json:"notify_url"`
	DescriptionPrefix     string    `json:"description_prefix"`
	TimeExpireMinutes     int       `json:"time_expire_minutes"`
	WechatPayPublicKeyID  string    `json:"wechatpay_public_key_id"`
	APIv3Key              string    `json:"apiv3_key"`
	WechatPayPublicKey    string    `json:"wechatpay_public_key"`
	KeyPem                string    `json:"key_pem"`
	HasAPIv3Key           bool      `json:"has_apiv3_key"`
	HasWechatPayPublicKey bool      `json:"has_wechatpay_public_key"`
	HasCertPem            bool      `json:"has_cert_pem"`
	HasKeyPem             bool      `json:"has_key_pem"`
	HasPlatformCert       bool      `json:"has_platform_cert"`
	ReadyForCheckout      bool      `json:"ready_for_checkout"`
	ValidationError       string    `json:"validation_error,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type SaveWechatPayConfigInput struct {
	MchID                   string `json:"mch_id"`
	AppID                   string `json:"app_id"`
	AuthMode                string `json:"auth_mode"`
	MerchantSerialNo        string `json:"merchant_serial_no"`
	APIv3Key                string `json:"apiv3_key"`
	ClearAPIv3Key           bool   `json:"clear_apiv3_key"`
	PlatformCertSerialNo    string `json:"platform_cert_serial_no"`
	NotifyURL               string `json:"notify_url"`
	DescriptionPrefix       string `json:"description_prefix"`
	TimeExpireMinutes       int    `json:"time_expire_minutes"`
	WechatPayPublicKeyID    string `json:"wechatpay_public_key_id"`
	WechatPayPublicKey      string `json:"wechatpay_public_key"`
	ClearWechatPayPublicKey bool   `json:"clear_wechatpay_public_key"`
	CertPem                 string `json:"cert_pem"`
	ClearCertPem            bool   `json:"clear_cert_pem"`
	KeyPem                  string `json:"key_pem"`
	ClearKeyPem             bool   `json:"clear_key_pem"`
	PlatformCert            string `json:"platform_cert"`
	ClearPlatformCert       bool   `json:"clear_platform_cert"`
}

type CreateWechatOrderInput struct {
	PlanID      uint   `json:"plan_id"`
	PlanKey     string `json:"plan_key"`
	SubjectKey  string `json:"subject_key"`
	CustomerRef string `json:"customer_ref"`
	Description string `json:"description"`
}

type WechatOrder struct {
	OrderNo         string     `json:"order_no"`
	PlanID          *uint      `json:"plan_id,omitempty"`
	PlanKey         string     `json:"plan_key"`
	SubjectKey      string     `json:"subject_key"`
	CustomerRef     string     `json:"customer_ref"`
	Description     string     `json:"description"`
	BillingMode     string     `json:"billing_mode"`
	AmountCents     int        `json:"amount_cents"`
	Currency        string     `json:"currency"`
	Provider        string     `json:"provider"`
	ProviderTradeNo string     `json:"provider_trade_no,omitempty"`
	CodeURL         string     `json:"code_url"`
	Status          string     `json:"status"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type SubscriptionStatus struct {
	ID                 uint       `json:"id"`
	CustomerRef        string     `json:"customer_ref"`
	PlanID             *uint      `json:"plan_id,omitempty"`
	PlanKey            string     `json:"plan_key"`
	SubjectKey         string     `json:"subject_key"`
	Status             string     `json:"status"`
	AutoRenew          bool       `json:"auto_renew"`
	Provider           string     `json:"provider"`
	ProviderContractID string     `json:"provider_contract_id,omitempty"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	CancelledAt        *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type PaymentOrderStatus struct {
	Order        WechatOrder         `json:"order"`
	Subscription *SubscriptionStatus `json:"subscription,omitempty"`
}

type PaymentOrderFilter struct {
	SubjectKey  string
	PlanKey     string
	CustomerRef string
	Status      string
	Query       string
	Page        int
	PageSize    int
}

type SubscriptionFilter struct {
	SubjectKey  string
	PlanKey     string
	CustomerRef string
	Status      string
	Query       string
	Page        int
	PageSize    int
}

type PagedPaymentOrders struct {
	Items    []WechatOrder `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type PagedSubscriptions struct {
	Items    []SubscriptionStatus `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type UpdateSubscriptionInput struct {
	PlanKey                 string `json:"plan_key"`
	Status                  string `json:"status"`
	AutoRenew               bool   `json:"auto_renew"`
	StartedAt               string `json:"started_at"`
	CurrentPeriodStart      string `json:"current_period_start"`
	CurrentPeriodEnd        string `json:"current_period_end"`
	CancelledAt             string `json:"cancelled_at"`
	ClearStartedAt          bool   `json:"clear_started_at"`
	ClearCurrentPeriodStart bool   `json:"clear_current_period_start"`
	ClearCurrentPeriodEnd   bool   `json:"clear_current_period_end"`
	ClearCancelledAt        bool   `json:"clear_cancelled_at"`
}
