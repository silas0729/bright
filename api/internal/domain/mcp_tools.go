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
	Query              string
	Category           string
	IsEnabled          *bool
	RequiresMembership *bool
	Page               int
	PageSize           int
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

func LocalizedMCPToolText(name, fallbackTitle, fallbackDescription string) (string, string) {
	switch name {
	case "list_subjects":
		return "查看学科列表", "列出 Brights 当前可用的全部学科。"
	case "list_categories":
		return "查看分类列表", "按学科和分类类型列出可用分类。"
	case "list_grades":
		return "查看阶段列表", "列出全部阶段与级别定义。"
	case "search_words":
		return "搜索单词", "按关键词和分页条件查询 Brights 单词库。"
	case "list_classification_stats":
		return "查看分类统计", "分页查看各分类下的词条统计数据。"
	case "list_membership_plans":
		return "查看会员方案", "列出 Brights 的会员方案和支付方案。"
	case "get_catalog_stats":
		return "查看词库总览", "获取 Brights 词库的整体统计数据。"
	case "search_knowledge_base":
		return "搜索知识库", "检索已上传的文本、表格或 Word 知识库内容。"
	case "list_my_knowledge_base_documents":
		return "查看我的知识库文档", "列出当前学员上传的文本、表格或 Word 知识库文档。"
	case "view_knowledge_base_document":
		return "查看知识库文档内容", "查看单个知识库文档的片段和原始内容。"
	case "list_my_payment_orders":
		return "查看我的订单", "列出当前学员的充值或购买订单记录。"
	case "list_my_memberships":
		return "查看我的会员", "列出当前学员的会员开通与续费记录。"
	case "get_invite_summary":
		return "查看邀请概览", "获取当前学员的邀请码和邀请统计。"
	case "get_my_invite_payout_profile":
		return "查看邀请提现资料", "获取当前学员的邀请佣金收款资料。"
	case "save_my_invite_payout_profile":
		return "保存邀请提现资料", "创建或更新当前学员的邀请佣金收款资料。"
	case "list_my_invite_commissions":
		return "查看邀请佣金", "列出当前学员的邀请佣金记录。"
	case "list_my_invite_withdraw_requests":
		return "查看提现申请", "列出当前学员的邀请佣金提现申请记录。"
	case "create_invite_withdraw_request":
		return "发起提现申请", "为当前学员的邀请佣金创建提现申请。"
	case "cancel_invite_withdraw_request":
		return "取消提现申请", "取消当前学员待处理的邀请佣金提现申请。"
	case "get_learning_summary":
		return "查看学习概况", "获取当前学员的等级分布、难度分布和记忆曲线统计。"
	case "list_learning_progress":
		return "查看学习进度", "列出当前学员已跟踪单词的等级、难度和下次复习时间。"
	case "save_learning_word_progress":
		return "保存单词进度", "创建或更新当前学员某个单词的学习等级和难度。"
	case "review_learning_word":
		return "记录单词复习", "记录当前学员是否记住该单词，并安排下一次复习。"
	case "xiaomi_get_devices":
		return "查看米家设备", "同步或列出当前学员绑定的小米 / 米家设备。"
	case "xiaomi_extract_tokens":
		return "刷新米家设备", "使用已保存凭证刷新当前学员的小米设备列表。"
	case "xiaomi_miot_prop_get":
		return "读取设备属性", "按 did、siid、piid 读取小米 MIoT 设备属性。"
	case "xiaomi_miot_prop_set":
		return "设置设备属性", "按 did、siid、piid 写入小米 MIoT 设备属性。"
	case "xiaomi_miot_action":
		return "执行设备动作", "按 did、siid、aiid 执行小米 MIoT 设备动作。"
	case "xiaomi_miot_prop_get_batch":
		return "批量读取属性", "批量读取多个小米 MIoT 设备属性。"
	case "xiaomi_find_device":
		return "查找米家设备", "按名称、型号、did、房间或家庭搜索小米设备。"
	case "xiaomi_control_device":
		return "语义控制设备", "按语义属性名或动作名控制小米设备。"
	case "list_mijia_homes":
		return "查看米家家庭", "列出当前学员小米账号下可用的家庭。"
	case "get_mijia_devices":
		return "查看家庭设备", "获取小米设备列表，可按家庭 ID 过滤。"
	case "get_device_status":
		return "查看设备状态", "读取小米设备的基础信息和可选 MIoT 属性状态。"
	case "control_device":
		return "控制设备", "以统一语义方式控制小米设备。"
	case "get_device_spec":
		return "查看设备规格", "拉取并缓存指定小米型号的 MIoT 规格。"
	case "mijia_list_devices":
		return "列出米家设备", "按条件列出小米设备，支持模糊搜索。"
	case "mijia_get_caps":
		return "查看设备能力", "根据 MIoT 规格汇总展示小米设备能力。"
	case "mijia_switch_set":
		return "开关设备", "控制小米开关类设备的开启与关闭。"
	case "mijia_sensor_get":
		return "读取传感器", "读取温度、湿度、电量等常见传感器数值。"
	case "mijia_position_set":
		return "设置位置值", "设置窗帘等支持位置控制的小米设备位置。"
	case "mijia_action_call":
		return "调用设备动作", "按动作名称执行小米设备的语义动作。"
	case "mijia_hvac_set":
		return "设置空调参数", "设置电源、模式、目标温度、风速等空调类小米控制项。"
	default:
		return fallbackTitle, fallbackDescription
	}
}

func DefaultMCPToolDefinitions() []MCPToolDefinition {
	items := []MCPToolDefinition{
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
			Description:               "Search uploaded text, spreadsheet, or Word knowledge base content.",
			Category:                  "knowledge",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_my_knowledge_base_documents",
			Title:                     "List My Knowledge Base Documents",
			Description:               "List the current learner's uploaded text, spreadsheet, or Word knowledge base documents.",
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
			Name:                      "get_my_invite_payout_profile",
			Title:                     "Get Invite Payout Profile",
			Description:               "Get the current learner's invite commission payout profile.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "save_my_invite_payout_profile",
			Title:                     "Save Invite Payout Profile",
			Description:               "Create or update the current learner's invite commission payout profile.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_my_invite_commissions",
			Title:                     "List Invite Commissions",
			Description:               "List the current learner's invite commission records.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "list_my_invite_withdraw_requests",
			Title:                     "List Invite Withdraw Requests",
			Description:               "List the current learner's invite commission withdraw requests.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "create_invite_withdraw_request",
			Title:                     "Create Invite Withdraw Request",
			Description:               "Create a withdraw request for the current learner's invite commissions.",
			Category:                  "account",
			SourceType:                "builtin",
			DefaultEnabled:            true,
			DefaultRequiresMembership: false,
		},
		{
			Name:                      "cancel_invite_withdraw_request",
			Title:                     "Cancel Invite Withdraw Request",
			Description:               "Cancel a pending invite commission withdraw request for the current learner.",
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
	for index := range items {
		items[index].Title, items[index].Description = LocalizedMCPToolText(
			items[index].Name,
			items[index].Title,
			items[index].Description,
		)
	}
	return items
}

func DefaultMCPToolDefinitionMap() map[string]MCPToolDefinition {
	items := DefaultMCPToolDefinitions()
	result := make(map[string]MCPToolDefinition, len(items))
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}
