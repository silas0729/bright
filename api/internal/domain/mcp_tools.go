package domain

import "time"

type MCPToolDefinition struct {
	Name                      string
	Title                     string
	Description               string
	Category                  string
	SourceType                string
	DefaultEnabled            bool
	DefaultRequiresMembership bool
}

type MCPToolConfig struct {
	ID                 uint      `json:"id"`
	ToolName           string    `json:"tool_name"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Category           string    `json:"category"`
	SourceType         string    `json:"source_type"`
	IsEnabled          bool      `json:"is_enabled"`
	RequiresMembership bool      `json:"requires_membership"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UpdateMCPToolConfigInput struct {
	IsEnabled          *bool `json:"is_enabled"`
	RequiresMembership *bool `json:"requires_membership"`
}

func DefaultMCPToolDefinitions() []MCPToolDefinition {
	return []MCPToolDefinition{
		{
			Name:                      "list_subjects",
			Title:                     "List Subjects",
			Description:               "List all Brights subjects.",
			Category:                  "catalog",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_categories",
			Title:                     "List Categories",
			Description:               "List categories for a subject and kind.",
			Category:                  "catalog",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_grades",
			Title:                     "List Grades",
			Description:               "List all grade definitions.",
			Category:                  "catalog",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "search_words",
			Title:                     "Search Words",
			Description:               "Search and paginate Brights words.",
			Category:                  "catalog",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_classification_stats",
			Title:                     "List Classification Stats",
			Description:               "List classification statistics with pagination.",
			Category:                  "catalog",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_membership_plans",
			Title:                     "List Membership Plans",
			Description:               "List Brights membership or payment plans.",
			Category:                  "membership",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "get_catalog_stats",
			Title:                     "Get Catalog Stats",
			Description:               "Get overall Brights catalog statistics.",
			Category:                  "catalog",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "search_knowledge_base",
			Title:                     "Search Knowledge Base",
			Description:               "Search uploaded text or spreadsheet knowledge base content.",
			Category:                  "knowledge",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_my_payment_orders",
			Title:                     "List My Payment Orders",
			Description:               "List the current learner's recharge or purchase records.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_my_memberships",
			Title:                     "List My Memberships",
			Description:               "List the current learner's membership records.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "get_invite_summary",
			Title:                     "Get Invite Summary",
			Description:               "Get the current learner's invite code and invite statistics.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
	}
}

func DefaultMCPToolDefinitionMap() map[string]MCPToolDefinition {
	items := DefaultMCPToolDefinitions()
	result := make(map[string]MCPToolDefinition, len(items))
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}
