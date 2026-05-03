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

type MCPToolConfigFilter struct {
	Query    string
	Category string
	Page     int
	PageSize int
}

type PagedMCPToolConfigs struct {
	Items    []MCPToolConfig `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
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
			Name:                      "list_my_knowledge_base_documents",
			Title:                     "List My Knowledge Base Documents",
			Description:               "List the current learner's uploaded knowledge base documents.",
			Category:                  "knowledge",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "view_knowledge_base_document",
			Title:                     "View Knowledge Base Document",
			Description:               "View chunks and original content for one uploaded knowledge base document.",
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
		{
			Name:                      "get_learning_summary",
			Title:                     "Get Learning Summary",
			Description:               "Get level counts, difficulty counts, and memory curve statistics for the current learner.",
			Category:                  "learning",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_learning_progress",
			Title:                     "List Learning Progress",
			Description:               "List tracked learning words with level, difficulty, and next review time.",
			Category:                  "learning",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "save_learning_word_progress",
			Title:                     "Save Learning Word Progress",
			Description:               "Create or update the current learner's level and difficulty for a word.",
			Category:                  "learning",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "review_learning_word",
			Title:                     "Review Learning Word",
			Description:               "Record whether the current learner remembered a word and schedule the next review.",
			Category:                  "learning",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_get_devices",
			Title:                     "Xiaomi Get Devices",
			Description:               "Sync or list Xiaomi / Mi Home devices for the current learner.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_extract_tokens",
			Title:                     "Xiaomi Refresh Devices",
			Description:               "Refresh the current learner's Xiaomi device list using stored credentials.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_miot_prop_get",
			Title:                     "Xiaomi MIoT Property Get",
			Description:               "Read a Xiaomi MIoT device property by did, siid, and piid.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_miot_prop_set",
			Title:                     "Xiaomi MIoT Property Set",
			Description:               "Write a Xiaomi MIoT device property by did, siid, and piid.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_miot_action",
			Title:                     "Xiaomi MIoT Action",
			Description:               "Execute a Xiaomi MIoT device action by did, siid, and aiid.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_miot_prop_get_batch",
			Title:                     "Xiaomi MIoT Property Get Batch",
			Description:               "Batch read multiple Xiaomi MIoT properties.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_find_device",
			Title:                     "Xiaomi Find Device",
			Description:               "Search a Xiaomi device by name, model, did, room, or home.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "xiaomi_control_device",
			Title:                     "Xiaomi Control Device",
			Description:               "Control a Xiaomi device by semantic property or action name.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_mijia_homes",
			Title:                     "List Mijia Homes",
			Description:               "List homes available in the learner's Xiaomi account.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "get_mijia_devices",
			Title:                     "Get Mijia Devices",
			Description:               "Get Xiaomi devices, optionally filtered by home id.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "get_device_status",
			Title:                     "Get Device Status",
			Description:               "Read device metadata and optional MIoT property status for a Xiaomi device.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "control_device",
			Title:                     "Control Device",
			Description:               "Semantic control wrapper for Xiaomi devices.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "get_device_spec",
			Title:                     "Get Device Spec",
			Description:               "Fetch and cache the MIoT spec for a Xiaomi model.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "mijia_list_devices",
			Title:                     "Mijia List Devices",
			Description:               "List Xiaomi devices with optional fuzzy search.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "mijia_get_caps",
			Title:                     "Mijia Get Caps",
			Description:               "Get summarized Xiaomi device capabilities from the MIoT spec.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "mijia_switch_set",
			Title:                     "Mijia Switch Set",
			Description:               "Turn a Xiaomi switch-like device on or off.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "mijia_sensor_get",
			Title:                     "Mijia Sensor Get",
			Description:               "Read common sensor values such as temperature, humidity, or battery.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "mijia_position_set",
			Title:                     "Mijia Position Set",
			Description:               "Set a Xiaomi curtain or position-capable device position.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "mijia_action_call",
			Title:                     "Mijia Action Call",
			Description:               "Execute a Xiaomi semantic action by action name.",
			Category:                  "smart-home",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "mijia_hvac_set",
			Title:                     "Mijia HVAC Set",
			Description:               "Set HVAC-like Xiaomi controls such as power, mode, target temperature, and fan level.",
			Category:                  "smart-home",
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
