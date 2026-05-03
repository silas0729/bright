package domain

import "time"

type APIConfig struct {
	ID                 uint      `json:"id"`
	Name               string    `json:"name"`
	ToolName           string    `json:"tool_name"`
	ResolvedToolName   string    `json:"resolved_tool_name"`
	URL                string    `json:"url"`
	Method             string    `json:"method"`
	Category           string    `json:"category"`
	CategoryColor      string    `json:"category_color"`
	Icon               string    `json:"icon"`
	Description        string    `json:"description"`
	Headers            string    `json:"headers"`
	Body               string    `json:"body"`
	Parameters         string    `json:"parameters"`
	IsActive           bool      `json:"is_active"`
	IsPublic           bool      `json:"is_public"`
	AllowAdminPublish  bool      `json:"allow_admin_publish"`
	OwnerLearnerUserID *uint     `json:"owner_learner_user_id,omitempty"`
	OwnerAdminUserID   *uint     `json:"owner_admin_user_id,omitempty"`
	OwnerName          string    `json:"owner_name,omitempty"`
	OwnerType          string    `json:"owner_type,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type APIConfigFilter struct {
	Query              string
	Category           string
	Page               int
	PageSize           int
	IncludeAll         bool
	OwnerLearnerUserID uint
	OwnerAdminUserID   uint
	PublicOnly         bool
}

type CreateAPIConfigInput struct {
	Name              string `json:"name"`
	ToolName          string `json:"tool_name"`
	URL               string `json:"url"`
	Method            string `json:"method"`
	Category          string `json:"category"`
	CategoryColor     string `json:"category_color"`
	Icon              string `json:"icon"`
	Description       string `json:"description"`
	Headers           string `json:"headers"`
	Body              string `json:"body"`
	Parameters        string `json:"parameters"`
	IsActive          bool   `json:"is_active"`
	IsPublic          *bool  `json:"is_public"`
	AllowAdminPublish *bool  `json:"allow_admin_publish"`
}

type UpdateAPIConfigInput struct {
	Name              string `json:"name"`
	ToolName          string `json:"tool_name"`
	URL               string `json:"url"`
	Method            string `json:"method"`
	Category          string `json:"category"`
	CategoryColor     string `json:"category_color"`
	Icon              string `json:"icon"`
	Description       string `json:"description"`
	Headers           string `json:"headers"`
	Body              string `json:"body"`
	Parameters        string `json:"parameters"`
	IsActive          bool   `json:"is_active"`
	IsPublic          *bool  `json:"is_public"`
	AllowAdminPublish *bool  `json:"allow_admin_publish"`
}

type PagedAPIConfigs struct {
	Items    []APIConfig `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type APIConfigTestInput struct {
	Arguments map[string]interface{} `json:"arguments"`
}

type APIConfigTestResult struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       interface{}       `json:"body,omitempty"`
	RawBody    string            `json:"raw_body,omitempty"`
}

type APIConfigExecutionContext struct {
	LearnerUserID uint
	Username      string
	SubjectKey    string
	AccessToken   string
	HTTPBaseURL   string
}

