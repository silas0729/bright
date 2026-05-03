package domain

import "time"

type MCPEndpoint struct {
	ID                uint      `json:"id"`
	LearnerUserID     uint      `json:"learner_user_id,omitempty"`
	Name              string    `json:"name"`
	URL               string    `json:"url"`
	Description       string    `json:"description"`
	Enabled           bool      `json:"enabled"`
	TokenQueryParam   string    `json:"token_query_param"`
	SubjectQueryParam string    `json:"subject_query_param"`
	ConnectionStatus  string    `json:"connection_status,omitempty"`
	IsConnected       bool      `json:"is_connected,omitempty"`
	LastError         string    `json:"last_error,omitempty"`
	ConnectedAt       *time.Time `json:"connected_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CreateMCPEndpointInput struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	Description       string `json:"description"`
	Enabled           bool   `json:"enabled"`
	TokenQueryParam   string `json:"token_query_param"`
	SubjectQueryParam string `json:"subject_query_param"`
}

type UpdateMCPEndpointInput struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	Description       string `json:"description"`
	Enabled           bool   `json:"enabled"`
	TokenQueryParam   string `json:"token_query_param"`
	SubjectQueryParam string `json:"subject_query_param"`
}
