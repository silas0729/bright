package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"brights/api/internal/domain"
	"brights/api/internal/service"
)

func xiaomiBuiltinTools() []Tool {
	return []Tool{
		{
			Name:        "xiaomi_get_devices",
			Title:       "Xiaomi Get Devices",
			Description: "Sync or list Xiaomi / Mi Home devices for the current learner.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"refresh": map[string]interface{}{"type": "boolean", "description": "Whether to refresh the cloud device list first."},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:         "xiaomi_extract_tokens",
			Title:        "Xiaomi Refresh Devices",
			Description:  "Refresh the current learner's Xiaomi device list using stored credentials.",
			Category:     "smart-home",
			SourceType:   "builtin",
			Enabled:      true,
			InputSchema:  objectSchema(nil),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "xiaomi_miot_prop_get",
			Title:       "Xiaomi MIoT Property Get",
			Description: "Read a Xiaomi MIoT device property.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":  map[string]interface{}{"type": "string"},
				"siid": map[string]interface{}{"type": "integer"},
				"piid": map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "xiaomi_miot_prop_set",
			Title:       "Xiaomi MIoT Property Set",
			Description: "Write a Xiaomi MIoT device property.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":   map[string]interface{}{"type": "string"},
				"siid":  map[string]interface{}{"type": "integer"},
				"piid":  map[string]interface{}{"type": "integer"},
				"value": map[string]interface{}{},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "xiaomi_miot_action",
			Title:       "Xiaomi MIoT Action",
			Description: "Execute a Xiaomi MIoT device action.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":  map[string]interface{}{"type": "string"},
				"siid": map[string]interface{}{"type": "integer"},
				"aiid": map[string]interface{}{"type": "integer"},
				"in":   map[string]interface{}{"type": "array"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "xiaomi_miot_prop_get_batch",
			Title:       "Xiaomi MIoT Property Get Batch",
			Description: "Batch read multiple Xiaomi MIoT properties.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"items": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"did":  map[string]interface{}{"type": "string"},
							"siid": map[string]interface{}{"type": "integer"},
							"piid": map[string]interface{}{"type": "integer"},
						},
					},
				},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "xiaomi_find_device",
			Title:       "Xiaomi Find Device",
			Description: "Search Xiaomi devices by query.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
				"q":     map[string]interface{}{"type": "string"},
				"limit": map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "xiaomi_control_device",
			Title:       "Xiaomi Control Device",
			Description: "Control a Xiaomi device using semantic property or action names.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"query":        map[string]interface{}{"type": "string"},
				"operation":    map[string]interface{}{"type": "string"},
				"prop_name":    map[string]interface{}{"type": "string"},
				"value":        map[string]interface{}{},
				"action_name":  map[string]interface{}{"type": "string"},
				"action_value": map[string]interface{}{},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:         "list_mijia_homes",
			Title:        "List Mijia Homes",
			Description:  "List Xiaomi homes for the current learner.",
			Category:     "smart-home",
			SourceType:   "builtin",
			Enabled:      true,
			InputSchema:  objectSchema(nil),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "get_mijia_devices",
			Title:       "Get Mijia Devices",
			Description: "Get Xiaomi devices, optionally filtered by home.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"refresh": map[string]interface{}{"type": "boolean"},
				"home_id": map[string]interface{}{"type": "string"},
				"query":   map[string]interface{}{"type": "string"},
				"limit":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "get_device_status",
			Title:       "Get Device Status",
			Description: "Read a Xiaomi device status with optional metadata.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":              map[string]interface{}{"type": "string"},
				"properties":       map[string]interface{}{"type": "array"},
				"include_metadata": map[string]interface{}{"type": "boolean"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "control_device",
			Title:       "Control Device",
			Description: "Semantic control wrapper for Xiaomi devices.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"device_id":    map[string]interface{}{"type": "string"},
				"device_name":  map[string]interface{}{"type": "string"},
				"operation":    map[string]interface{}{"type": "string"},
				"prop_name":    map[string]interface{}{"type": "string"},
				"value":        map[string]interface{}{},
				"action_name":  map[string]interface{}{"type": "string"},
				"action_value": map[string]interface{}{},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "get_device_spec",
			Title:       "Get Device Spec",
			Description: "Fetch and cache the MIoT spec by model.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"model": map[string]interface{}{"type": "string"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "mijia_list_devices",
			Title:       "Mijia List Devices",
			Description: "List Xiaomi devices with optional fuzzy search.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
				"limit": map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "mijia_get_caps",
			Title:       "Mijia Get Caps",
			Description: "Get summarized Xiaomi device capabilities from the MIoT spec.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":     map[string]interface{}{"type": "string"},
				"verbose": map[string]interface{}{"type": "boolean"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "mijia_switch_set",
			Title:       "Mijia Switch Set",
			Description: "Turn a Xiaomi switch-like device on or off.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did": map[string]interface{}{"type": "string"},
				"on":  map[string]interface{}{"type": "boolean"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "mijia_sensor_get",
			Title:       "Mijia Sensor Get",
			Description: "Read common sensor values such as temperature or humidity.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did": map[string]interface{}{"type": "string"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "mijia_position_set",
			Title:       "Mijia Position Set",
			Description: "Set a Xiaomi curtain or position-capable device position.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":      map[string]interface{}{"type": "string"},
				"position": map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "mijia_action_call",
			Title:       "Mijia Action Call",
			Description: "Execute a Xiaomi semantic action by action name.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":    map[string]interface{}{"type": "string"},
				"action": map[string]interface{}{"type": "string"},
				"in":     map[string]interface{}{"type": "array"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "mijia_hvac_set",
			Title:       "Mijia HVAC Set",
			Description: "Set HVAC-like Xiaomi controls such as power, mode, target temperature, and fan level.",
			Category:    "smart-home",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"did":    map[string]interface{}{"type": "string"},
				"params": map[string]interface{}{"type": "object"},
			}),
			OutputSchema: toolResultSchema,
		},
	}
}

func (s *Server) dynamicAPITools(ctx context.Context, session *Session) []Tool {
	learnerID := uint(0)
	if session != nil {
		learnerID = session.UserID
	}

	items, err := s.service.ListAccessibleAPIConfigs(ctx, learnerID)
	if err != nil {
		return nil
	}

	tools := make([]Tool, 0, len(items))
	for _, item := range items {
		tools = append(tools, Tool{
			Name:         item.ResolvedToolName,
			Title:        firstNonEmpty(item.Name, item.ResolvedToolName),
			Description:  firstNonEmpty(item.Description, fmt.Sprintf("Dynamic API tool for %s", item.Name)),
			Category:     firstNonEmpty(item.Category, "api"),
			SourceType:   apiToolSourceType(item),
			Enabled:      item.IsActive,
			InputSchema:  service.BuildAPIConfigInputSchema(item),
			OutputSchema: toolResultSchema,
		})
	}
	return tools
}

func (s *Server) handleXiaomiTool(ctx context.Context, session Session, req CallToolRequest) (CallToolResult, bool, error) {
	switch canonicalToolName(req.Name) {
	case "xiaomi_get_devices":
		data, err := s.service.ListLearnerXiaomiDevices(ctx, session.UserID, boolArg(req.Arguments, "refresh", false))
		return handledToolResult("xiaomi_get_devices", data, err)
	case "xiaomi_extract_tokens":
		data, err := s.service.RefreshLearnerXiaomiDevices(ctx, session.UserID)
		return handledToolResult("xiaomi_extract_tokens", map[string]interface{}{
			"device_count": len(data),
			"devices":      data,
		}, err)
	case "xiaomi_miot_prop_get":
		data, err := s.service.LearnerXiaomiPropGet(ctx, session.UserID, domain.XiaomiPropGetInput{
			Did:  stringArg(req.Arguments, "did", ""),
			Siid: int64(intArg(req.Arguments, "siid", 0)),
			Piid: int64(intArg(req.Arguments, "piid", 0)),
		})
		return handledToolResult("xiaomi_miot_prop_get", data, err)
	case "xiaomi_miot_prop_set":
		data, err := s.service.LearnerXiaomiPropSet(ctx, session.UserID, domain.XiaomiPropSetInput{
			Did:   stringArg(req.Arguments, "did", ""),
			Siid:  int64(intArg(req.Arguments, "siid", 0)),
			Piid:  int64(intArg(req.Arguments, "piid", 0)),
			Value: req.Arguments["value"],
		})
		return handledToolResult("xiaomi_miot_prop_set", data, err)
	case "xiaomi_miot_action":
		in := interfaceSliceArg(req.Arguments, "in")
		data, err := s.service.LearnerXiaomiAction(ctx, session.UserID, domain.XiaomiActionInput{
			Did:  stringArg(req.Arguments, "did", ""),
			Siid: int64(intArg(req.Arguments, "siid", 0)),
			Aiid: int64(intArg(req.Arguments, "aiid", 0)),
			In:   in,
		})
		return handledToolResult("xiaomi_miot_action", data, err)
	case "xiaomi_miot_prop_get_batch":
		rawItems, _ := req.Arguments["items"].([]interface{})
		items := make([]domain.XiaomiBatchPropItem, 0, len(rawItems))
		for _, rawItem := range rawItems {
			itemMap, _ := rawItem.(map[string]interface{})
			items = append(items, domain.XiaomiBatchPropItem{
				Did:  stringArg(itemMap, "did", ""),
				Siid: int64(intArg(itemMap, "siid", 0)),
				Piid: int64(intArg(itemMap, "piid", 0)),
			})
		}
		data, err := s.service.LearnerXiaomiPropGetBatch(ctx, session.UserID, items)
		return handledToolResult("xiaomi_miot_prop_get_batch", data, err)
	case "xiaomi_find_device":
		data, err := s.service.FindLearnerXiaomiDevices(
			ctx,
			session.UserID,
			firstNonEmpty(stringArg(req.Arguments, "query", ""), stringArg(req.Arguments, "q", "")),
			intArg(req.Arguments, "limit", 10),
		)
		return handledToolResult("xiaomi_find_device", map[string]interface{}{
			"results": data,
			"total":   len(data),
		}, err)
	case "xiaomi_control_device":
		data, err := s.service.ControlLearnerXiaomiDevice(ctx, session.UserID, domain.XiaomiControlDeviceInput{
			Query:       stringArg(req.Arguments, "query", ""),
			Operation:   stringArg(req.Arguments, "operation", "set_property"),
			PropName:    stringArg(req.Arguments, "prop_name", ""),
			Value:       req.Arguments["value"],
			ActionName:  stringArg(req.Arguments, "action_name", ""),
			ActionValue: req.Arguments["action_value"],
		})
		return handledToolResult("xiaomi_control_device", data, err)
	case "list_mijia_homes":
		data, err := s.service.ListLearnerXiaomiHomes(ctx, session.UserID)
		return handledToolResult("list_mijia_homes", data, err)
	case "get_mijia_devices":
		data, err := s.service.ListLearnerXiaomiDevices(ctx, session.UserID, boolArg(req.Arguments, "refresh", false))
		if err != nil {
			return CallToolResult{}, true, err
		}
		homeID := stringArg(req.Arguments, "home_id", "")
		query := strings.ToLower(strings.TrimSpace(stringArg(req.Arguments, "query", "")))
		limit := intArg(req.Arguments, "limit", 200)
		items := make([]domain.XiaomiDevice, 0, len(data.Devices))
		for _, item := range data.Devices {
			if homeID != "" && item.HomeID != homeID {
				continue
			}
			if query != "" &&
				!strings.Contains(strings.ToLower(item.Name), query) &&
				!strings.Contains(strings.ToLower(item.Model), query) &&
				!strings.Contains(strings.ToLower(item.Did), query) {
				continue
			}
			items = append(items, item)
			if limit > 0 && len(items) >= limit {
				break
			}
		}
		return handledToolResult("get_mijia_devices", items, nil)
	case "get_device_status":
		var propertyNames []string
		for _, value := range interfaceSliceArg(req.Arguments, "properties") {
			propertyNames = append(propertyNames, strings.TrimSpace(fmt.Sprintf("%v", value)))
		}
		data, err := s.service.GetLearnerXiaomiDeviceStatus(
			ctx,
			session.UserID,
			stringArg(req.Arguments, "did", ""),
			propertyNames,
			boolArg(req.Arguments, "include_metadata", true),
		)
		return handledToolResult("get_device_status", data, err)
	case "control_device":
		query := firstNonEmpty(stringArg(req.Arguments, "device_id", ""), stringArg(req.Arguments, "device_name", ""))
		data, err := s.service.ControlLearnerXiaomiDevice(ctx, session.UserID, domain.XiaomiControlDeviceInput{
			Query:       query,
			Operation:   stringArg(req.Arguments, "operation", "set_property"),
			PropName:    stringArg(req.Arguments, "prop_name", ""),
			Value:       req.Arguments["value"],
			ActionName:  stringArg(req.Arguments, "action_name", ""),
			ActionValue: req.Arguments["action_value"],
		})
		return handledToolResult("control_device", data, err)
	case "get_device_spec":
		spec, raw, err := s.service.GetLearnerMiotSpec(ctx, stringArg(req.Arguments, "model", ""))
		if err != nil {
			return CallToolResult{}, true, err
		}
		var parsed interface{}
		_ = json.Unmarshal(raw, &parsed)
		return handledToolResult("get_device_spec", map[string]interface{}{
			"spec":    parsed,
			"summary": buildMiotSpecSummary(spec),
		}, nil)
	case "mijia_list_devices":
		data, err := s.service.ListLearnerXiaomiDevices(ctx, session.UserID, false)
		if err != nil {
			return CallToolResult{}, true, err
		}
		query := strings.ToLower(strings.TrimSpace(stringArg(req.Arguments, "query", "")))
		limit := intArg(req.Arguments, "limit", 50)
		items := make([]domain.XiaomiDevice, 0, len(data.Devices))
		for _, item := range data.Devices {
			if query != "" && !strings.Contains(strings.ToLower(item.Name), query) && !strings.Contains(strings.ToLower(item.Model), query) {
				continue
			}
			items = append(items, item)
			if limit > 0 && len(items) >= limit {
				break
			}
		}
		return handledToolResult("mijia_list_devices", items, nil)
	case "mijia_get_caps":
		spec, _, err := s.service.GetLearnerMiotSpec(ctx, stringArg(req.Arguments, "model", ""))
		if err == nil && stringArg(req.Arguments, "did", "") == "" {
			return handledToolResult("mijia_get_caps", buildMiotSpecSummary(spec), nil)
		}
		data, err := s.service.GetLearnerXiaomiDeviceStatus(
			ctx,
			session.UserID,
			stringArg(req.Arguments, "did", ""),
			nil,
			true,
		)
		return handledToolResult("mijia_get_caps", data, err)
	case "mijia_switch_set":
		data, err := s.service.MijiaSwitchSet(ctx, session.UserID, stringArg(req.Arguments, "did", ""), boolArg(req.Arguments, "on", false))
		return handledToolResult("mijia_switch_set", data, err)
	case "mijia_sensor_get":
		data, err := s.service.MijiaSensorGet(ctx, session.UserID, stringArg(req.Arguments, "did", ""))
		return handledToolResult("mijia_sensor_get", data, err)
	case "mijia_position_set":
		data, err := s.service.MijiaPositionSet(ctx, session.UserID, stringArg(req.Arguments, "did", ""), intArg(req.Arguments, "position", 0))
		return handledToolResult("mijia_position_set", data, err)
	case "mijia_action_call":
		data, err := s.service.MijiaActionCall(
			ctx,
			session.UserID,
			stringArg(req.Arguments, "did", ""),
			stringArg(req.Arguments, "action", ""),
			interfaceSliceArg(req.Arguments, "in"),
		)
		return handledToolResult("mijia_action_call", data, err)
	case "mijia_hvac_set":
		paramsMap, _ := req.Arguments["params"].(map[string]interface{})
		data, err := s.service.MijiaHvacSet(ctx, session.UserID, stringArg(req.Arguments, "did", ""), paramsMap)
		return handledToolResult("mijia_hvac_set", data, err)
	default:
		return CallToolResult{}, false, nil
	}
}

func (s *Server) callDynamicAPITool(ctx context.Context, session Session, req CallToolRequest) (CallToolResult, bool, error) {
	result, err := s.service.ExecuteAccessibleAPIConfigByToolName(ctx, session.UserID, req.Name, req.Arguments, domain.APIConfigExecutionContext{
		LearnerUserID: session.UserID,
		Username:      session.Username,
		SubjectKey:    session.SubjectKey,
		AccessToken:   session.Token,
		HTTPBaseURL:   session.HTTPBaseURL,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			return CallToolResult{}, false, nil
		}
		return CallToolResult{}, true, err
	}
	payload := map[string]interface{}{
		"success":     result.StatusCode >= 200 && result.StatusCode < 400,
		"tool":        canonicalToolName(req.Name),
		"status_code": result.StatusCode,
		"headers":     result.Headers,
	}
	if result.Body != nil {
		payload["result"] = result.Body
	} else {
		payload["result"] = result.RawBody
	}
	textBytes, _ := json.MarshalIndent(payload, "", "  ")
	return CallToolResult{
		StructuredContent: payload,
		Content: []Content{{
			Type: "text",
			Text: string(textBytes),
		}},
		IsError: !(result.StatusCode >= 200 && result.StatusCode < 400),
	}, true, nil
}

func apiToolSourceType(item domain.APIConfig) string {
	if item.OwnerLearnerUserID != nil {
		return "api_config_personal"
	}
	return "api_config"
}

func handledToolResult(toolName string, data interface{}, err error) (CallToolResult, bool, error) {
	result, callErr := newToolResult(toolName, data, err)
	return result, true, callErr
}

func buildMiotSpecSummary(spec domain.MiotSpecParsed) map[string]interface{} {
	properties := make([]map[string]interface{}, 0, len(spec.Properties))
	for _, property := range spec.Properties {
		properties = append(properties, map[string]interface{}{
			"name":  property.Name,
			"desc":  property.Description,
			"type":  property.Type,
			"rw":    property.RW,
			"siid":  property.Method.SIID,
			"piid":  property.Method.PIID,
			"range": property.Range,
			"enum":  property.ValueList,
		})
	}
	actions := make([]map[string]interface{}, 0, len(spec.Actions))
	for _, action := range spec.Actions {
		actions = append(actions, map[string]interface{}{
			"name": action.Name,
			"desc": action.Description,
			"siid": action.Method.SIID,
			"aiid": action.Method.AIID,
		})
	}
	return map[string]interface{}{
		"name":       spec.Name,
		"model":      spec.Model,
		"properties": properties,
		"actions":    actions,
	}
}

func boolArg(args map[string]interface{}, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		switch trimmed {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	case float64:
		return typed != 0
	case int:
		return typed != 0
	}
	return fallback
}

func interfaceSliceArg(args map[string]interface{}, key string) []interface{} {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	return items
}
