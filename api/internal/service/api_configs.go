package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"brights/api/internal/domain"
	"brights/api/internal/storage"

	"gorm.io/gorm"
)

const apiConfigSourceType = "api_config"

var apiTemplatePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

type apiParameterDefinition struct {
	Name        string
	Type        string
	In          string
	Description string
	Required    bool
}

func (s *Service) ListAdminAPIConfigs(ctx context.Context, filter domain.APIConfigFilter) (domain.PagedAPIConfigs, error) {
	filter.IncludeAll = true
	return s.listAPIConfigs(ctx, filter)
}

func (s *Service) ListLearnerAPIConfigs(ctx context.Context, learnerID uint, filter domain.APIConfigFilter) (domain.PagedAPIConfigs, error) {
	if learnerID == 0 {
		return domain.PagedAPIConfigs{}, errors.New("learner id is required")
	}
	filter.OwnerLearnerUserID = learnerID
	return s.listAPIConfigs(ctx, filter)
}

func (s *Service) ListAccessibleAPIConfigs(ctx context.Context, learnerID uint) ([]domain.APIConfig, error) {
	query := s.db.WithContext(ctx).
		Model(&storage.APIConfig{}).
		Where("is_active = ?", true)
	if learnerID > 0 {
		query = query.Where("(is_public = ? OR owner_learner_user_id = ?)", true, learnerID)
	} else {
		query = query.Where("is_public = ?", true)
	}

	var models []storage.APIConfig
	if err := query.Order("category asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}

	return s.toAPIConfigItems(ctx, models)
}

func (s *Service) CreateAdminAPIConfig(ctx context.Context, adminID uint, input domain.CreateAPIConfigInput) (domain.APIConfig, error) {
	if adminID == 0 {
		return domain.APIConfig{}, errors.New("admin id is required")
	}
	return s.createAPIConfig(ctx, nil, uintPtr(adminID), input)
}

func (s *Service) CreateLearnerAPIConfig(ctx context.Context, learnerID uint, input domain.CreateAPIConfigInput) (domain.APIConfig, error) {
	if learnerID == 0 {
		return domain.APIConfig{}, errors.New("learner id is required")
	}
	return s.createAPIConfig(ctx, uintPtr(learnerID), nil, input)
}

func (s *Service) UpdateAdminAPIConfig(ctx context.Context, adminID, id uint, input domain.UpdateAPIConfigInput) (domain.APIConfig, error) {
	if adminID == 0 {
		return domain.APIConfig{}, errors.New("admin id is required")
	}
	return s.updateAPIConfig(ctx, id, nil, input)
}

func (s *Service) UpdateLearnerAPIConfig(ctx context.Context, learnerID, id uint, input domain.UpdateAPIConfigInput) (domain.APIConfig, error) {
	if learnerID == 0 {
		return domain.APIConfig{}, errors.New("learner id is required")
	}
	return s.updateAPIConfig(ctx, id, uintPtr(learnerID), input)
}

func (s *Service) DeleteAdminAPIConfig(ctx context.Context, adminID, id uint) error {
	if adminID == 0 {
		return errors.New("admin id is required")
	}
	return s.deleteAPIConfig(ctx, id, nil)
}

func (s *Service) DeleteLearnerAPIConfig(ctx context.Context, learnerID, id uint) error {
	if learnerID == 0 {
		return errors.New("learner id is required")
	}
	return s.deleteAPIConfig(ctx, id, uintPtr(learnerID))
}

func (s *Service) TestAdminAPIConfig(
	ctx context.Context,
	adminID, id uint,
	input domain.APIConfigTestInput,
	execCtx domain.APIConfigExecutionContext,
) (domain.APIConfigTestResult, error) {
	if adminID == 0 {
		return domain.APIConfigTestResult{}, errors.New("admin id is required")
	}
	model, err := s.findAPIConfigForActor(ctx, id, nil)
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}
	return s.executeAPIConfigModel(ctx, model, input.Arguments, execCtx)
}

func (s *Service) TestLearnerAPIConfig(
	ctx context.Context,
	learnerID, id uint,
	input domain.APIConfigTestInput,
	execCtx domain.APIConfigExecutionContext,
) (domain.APIConfigTestResult, error) {
	if learnerID == 0 {
		return domain.APIConfigTestResult{}, errors.New("learner id is required")
	}
	model, err := s.findAPIConfigForActor(ctx, id, uintPtr(learnerID))
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}
	return s.executeAPIConfigModel(ctx, model, input.Arguments, execCtx)
}

func (s *Service) FindAccessibleAPIConfigByToolName(ctx context.Context, learnerID uint, toolName string) (domain.APIConfig, error) {
	items, err := s.ListAccessibleAPIConfigs(ctx, learnerID)
	if err != nil {
		return domain.APIConfig{}, err
	}
	normalized := normalizeToolName(toolName)
	for _, item := range items {
		if normalizeToolName(item.ResolvedToolName) == normalized {
			return item, nil
		}
	}
	return domain.APIConfig{}, errors.New("api config tool does not exist")
}

func (s *Service) ExecuteAccessibleAPIConfigByToolName(
	ctx context.Context,
	learnerID uint,
	toolName string,
	args map[string]interface{},
	execCtx domain.APIConfigExecutionContext,
) (domain.APIConfigTestResult, error) {
	model, err := s.findAccessibleAPIConfigModelByToolName(ctx, learnerID, toolName)
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}
	return s.executeAPIConfigModel(ctx, model, args, execCtx)
}

func BuildAPIConfigInputSchema(config domain.APIConfig) map[string]interface{} {
	properties := map[string]interface{}{}
	required := make([]string, 0, 8)

	definitions, err := parseAPIParameterDefinitions(config.Parameters)
	if err == nil {
		for _, definition := range definitions {
			schemaType := normalizeAPIParameterType(definition.Type)
			properties[definition.Name] = map[string]interface{}{
				"type":        schemaType,
				"description": strings.TrimSpace(definition.Description),
			}
			if definition.Required {
				required = append(required, definition.Name)
			}
		}
	}

	for _, key := range extractAPITemplateKeys(config.URL, config.Headers, config.Body) {
		if _, exists := properties[key]; exists {
			continue
		}
		properties[key] = map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("Template variable: %s", key),
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func (s *Service) listAPIConfigs(ctx context.Context, filter domain.APIConfigFilter) (domain.PagedAPIConfigs, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)

	query := s.db.WithContext(ctx).Model(&storage.APIConfig{})
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR tool_name LIKE ? OR category LIKE ? OR description LIKE ?", like, like, like, like)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		query = query.Where("category = ?", category)
	}

	switch {
	case filter.IncludeAll:
	case filter.OwnerLearnerUserID > 0:
		query = query.Where("owner_learner_user_id = ?", filter.OwnerLearnerUserID)
	case filter.OwnerAdminUserID > 0:
		query = query.Where("owner_admin_user_id = ?", filter.OwnerAdminUserID)
	case filter.PublicOnly:
		query = query.Where("is_public = ?", true)
	default:
		query = query.Where("1 = 0")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedAPIConfigs{}, err
	}

	var models []storage.APIConfig
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedAPIConfigs{}, err
	}

	items, err := s.toAPIConfigItems(ctx, models)
	if err != nil {
		return domain.PagedAPIConfigs{}, err
	}

	return domain.PagedAPIConfigs{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) createAPIConfig(
	ctx context.Context,
	ownerLearnerUserID *uint,
	ownerAdminUserID *uint,
	input domain.CreateAPIConfigInput,
) (domain.APIConfig, error) {
	validated, err := validateAPIConfigInput(input.Name, input.ToolName, input.URL, input.Method, input.Headers, input.Body, input.Parameters)
	if err != nil {
		return domain.APIConfig{}, err
	}

	model := storage.APIConfig{
		Name:               validated.Name,
		ToolName:           validated.ToolName,
		URL:                validated.URL,
		Method:             validated.Method,
		Category:           strings.TrimSpace(input.Category),
		CategoryColor:      firstNonEmpty(strings.TrimSpace(input.CategoryColor), "primary"),
		Icon:               firstNonEmpty(strings.TrimSpace(input.Icon), "bi-plug"),
		Description:        strings.TrimSpace(input.Description),
		Headers:            strings.TrimSpace(input.Headers),
		Body:               strings.TrimSpace(input.Body),
		Parameters:         strings.TrimSpace(input.Parameters),
		IsActive:           input.IsActive,
		IsPublic:           false,
		AllowAdminPublish:  false,
		OwnerLearnerUserID: ownerLearnerUserID,
		OwnerAdminUserID:   ownerAdminUserID,
	}
	if ownerAdminUserID != nil {
		model.IsPublic = true
		model.AllowAdminPublish = true
	}
	if input.IsPublic != nil {
		model.IsPublic = *input.IsPublic
	}
	if input.AllowAdminPublish != nil {
		model.AllowAdminPublish = *input.AllowAdminPublish
	}

	if model.ToolName != "" {
		if err := s.ensureToolNameIsAvailable(ctx, model.ToolName, 0, ""); err != nil {
			return domain.APIConfig{}, err
		}
	}

	var created storage.APIConfig
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		if err := s.ensureToolNameIsAvailableTx(tx, resolvedAPIConfigToolName(model), model.ID, ""); err != nil {
			return err
		}
		if err := s.syncAPIConfigToolConfigTx(tx, model, ""); err != nil {
			return err
		}
		if err := tx.Where("id = ?", model.ID).First(&created).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return domain.APIConfig{}, err
	}

	items, err := s.toAPIConfigItems(ctx, []storage.APIConfig{created})
	if err != nil {
		return domain.APIConfig{}, err
	}
	return items[0], nil
}

func (s *Service) updateAPIConfig(
	ctx context.Context,
	id uint,
	learnerID *uint,
	input domain.UpdateAPIConfigInput,
) (domain.APIConfig, error) {
	if id == 0 {
		return domain.APIConfig{}, errors.New("api config id is required")
	}

	model, err := s.findAPIConfigForActor(ctx, id, learnerID)
	if err != nil {
		return domain.APIConfig{}, err
	}
	previousResolvedToolName := resolvedAPIConfigToolName(model)

	validated, err := validateAPIConfigInput(input.Name, input.ToolName, input.URL, input.Method, input.Headers, input.Body, input.Parameters)
	if err != nil {
		return domain.APIConfig{}, err
	}

	model.Name = validated.Name
	model.ToolName = validated.ToolName
	model.URL = validated.URL
	model.Method = validated.Method
	model.Category = strings.TrimSpace(input.Category)
	model.CategoryColor = firstNonEmpty(strings.TrimSpace(input.CategoryColor), "primary")
	model.Icon = firstNonEmpty(strings.TrimSpace(input.Icon), "bi-plug")
	model.Description = strings.TrimSpace(input.Description)
	model.Headers = strings.TrimSpace(input.Headers)
	model.Body = strings.TrimSpace(input.Body)
	model.Parameters = strings.TrimSpace(input.Parameters)
	model.IsActive = input.IsActive
	if input.IsPublic != nil {
		model.IsPublic = *input.IsPublic
	}
	if input.AllowAdminPublish != nil {
		model.AllowAdminPublish = *input.AllowAdminPublish
	}

	if model.ToolName != "" {
		if err := s.ensureToolNameIsAvailable(ctx, model.ToolName, model.ID, previousResolvedToolName); err != nil {
			return domain.APIConfig{}, err
		}
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.ensureToolNameIsAvailableTx(tx, resolvedAPIConfigToolName(model), model.ID, previousResolvedToolName); err != nil {
			return err
		}
		if err := tx.Save(&model).Error; err != nil {
			return err
		}
		return s.syncAPIConfigToolConfigTx(tx, model, previousResolvedToolName)
	}); err != nil {
		return domain.APIConfig{}, err
	}

	items, err := s.toAPIConfigItems(ctx, []storage.APIConfig{model})
	if err != nil {
		return domain.APIConfig{}, err
	}
	return items[0], nil
}

func (s *Service) deleteAPIConfig(ctx context.Context, id uint, learnerID *uint) error {
	if id == 0 {
		return errors.New("api config id is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model, err := s.findAPIConfigForActorTx(tx, id, learnerID)
		if err != nil {
			return err
		}
		resolvedToolName := resolvedAPIConfigToolName(model)
		if err := tx.Delete(&model).Error; err != nil {
			return err
		}
		return tx.Where("tool_name = ?", resolvedToolName).Delete(&storage.MCPToolConfig{}).Error
	})
}

func (s *Service) executeAPIConfigModel(
	ctx context.Context,
	model storage.APIConfig,
	args map[string]interface{},
	execCtx domain.APIConfigExecutionContext,
) (domain.APIConfigTestResult, error) {
	if !model.IsActive {
		return domain.APIConfigTestResult{}, errors.New("api config is disabled")
	}

	if args == nil {
		args = map[string]interface{}{}
	}

	resolvedURL, err := resolveAPIConfigURL(model.URL, execCtx, args)
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}

	method := strings.ToUpper(strings.TrimSpace(model.Method))
	if method == "" {
		method = http.MethodGet
	}

	headers, err := renderAPIConfigHeaders(model.Headers, execCtx, args)
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}
	if headers == nil {
		headers = map[string]string{}
	}

	if execCtx.AccessToken != "" && strings.HasPrefix(strings.TrimSpace(model.URL), "/api/v1/auth/") {
		if _, exists := headers["Authorization"]; !exists {
			headers["Authorization"] = "Bearer " + execCtx.AccessToken
		}
	}

	var bodyReader io.Reader
	if method != http.MethodGet && method != http.MethodHead {
		payload, contentType, err := renderAPIConfigBody(model.Body, execCtx, args)
		if err != nil {
			return domain.APIConfigTestResult{}, err
		}
		if len(payload) == 0 {
			payload, err = json.Marshal(args)
			if err != nil {
				return domain.APIConfigTestResult{}, err
			}
			if len(args) == 0 {
				payload = nil
			}
			if len(payload) > 0 && contentType == "" {
				contentType = "application/json"
			}
		}
		if len(payload) > 0 {
			bodyReader = bytes.NewReader(payload)
			if contentType != "" && headers["Content-Type"] == "" {
				headers["Content-Type"] = contentType
			}
		}
	}

	if method == http.MethodGet || method == http.MethodDelete || method == http.MethodHead {
		augmentedURL, err := appendAPIConfigQueryArgs(resolvedURL, model, execCtx, args)
		if err != nil {
			return domain.APIConfigTestResult{}, err
		}
		resolvedURL = augmentedURL
	}

	request, err := http.NewRequestWithContext(ctx, method, resolvedURL, bodyReader)
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		request.Header.Set(key, value)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.APIConfigTestResult{}, err
	}

	headerMap := make(map[string]string, len(response.Header))
	for key, values := range response.Header {
		headerMap[key] = strings.Join(values, ", ")
	}

	rawBody := string(bodyBytes)
	result := domain.APIConfigTestResult{
		StatusCode: response.StatusCode,
		Headers:    headerMap,
		RawBody:    rawBody,
	}
	var parsed interface{}
	if err := json.Unmarshal(bodyBytes, &parsed); err == nil {
		result.Body = parsed
	}
	return result, nil
}

func (s *Service) toAPIConfigItems(ctx context.Context, models []storage.APIConfig) ([]domain.APIConfig, error) {
	learnerIDs := make([]uint, 0, len(models))
	adminIDs := make([]uint, 0, len(models))
	learnerSeen := make(map[uint]struct{}, len(models))
	adminSeen := make(map[uint]struct{}, len(models))
	for _, model := range models {
		if model.OwnerLearnerUserID != nil {
			if _, exists := learnerSeen[*model.OwnerLearnerUserID]; !exists {
				learnerSeen[*model.OwnerLearnerUserID] = struct{}{}
				learnerIDs = append(learnerIDs, *model.OwnerLearnerUserID)
			}
		}
		if model.OwnerAdminUserID != nil {
			if _, exists := adminSeen[*model.OwnerAdminUserID]; !exists {
				adminSeen[*model.OwnerAdminUserID] = struct{}{}
				adminIDs = append(adminIDs, *model.OwnerAdminUserID)
			}
		}
	}

	learnerMap := map[uint]string{}
	if len(learnerIDs) > 0 {
		var learners []storage.LearnerUser
		if err := s.db.WithContext(ctx).Select("id", "username", "display_name").Where("id IN ?", learnerIDs).Find(&learners).Error; err != nil {
			return nil, err
		}
		for _, learner := range learners {
			learnerMap[learner.ID] = firstNonEmpty(learner.DisplayName, learner.Username)
		}
	}

	adminMap := map[uint]string{}
	if len(adminIDs) > 0 {
		var admins []storage.AdminUser
		if err := s.db.WithContext(ctx).Select("id", "username", "display_name").Where("id IN ?", adminIDs).Find(&admins).Error; err != nil {
			return nil, err
		}
		for _, admin := range admins {
			adminMap[admin.ID] = firstNonEmpty(admin.DisplayName, admin.Username)
		}
	}

	items := make([]domain.APIConfig, 0, len(models))
	for _, model := range models {
		item := domain.APIConfig{
			ID:                 model.ID,
			Name:               model.Name,
			ToolName:           model.ToolName,
			ResolvedToolName:   resolvedAPIConfigToolName(model),
			URL:                model.URL,
			Method:             model.Method,
			Category:           model.Category,
			CategoryColor:      model.CategoryColor,
			Icon:               model.Icon,
			Description:        model.Description,
			Headers:            model.Headers,
			Body:               model.Body,
			Parameters:         model.Parameters,
			IsActive:           model.IsActive,
			IsPublic:           model.IsPublic,
			AllowAdminPublish:  model.AllowAdminPublish,
			OwnerLearnerUserID: model.OwnerLearnerUserID,
			OwnerAdminUserID:   model.OwnerAdminUserID,
			CreatedAt:          model.CreatedAt,
			UpdatedAt:          model.UpdatedAt,
		}
		if model.OwnerLearnerUserID != nil {
			item.OwnerType = "learner"
			item.OwnerName = learnerMap[*model.OwnerLearnerUserID]
		}
		if model.OwnerAdminUserID != nil {
			item.OwnerType = "admin"
			item.OwnerName = adminMap[*model.OwnerAdminUserID]
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) findAccessibleAPIConfigModelByToolName(ctx context.Context, learnerID uint, toolName string) (storage.APIConfig, error) {
	var models []storage.APIConfig
	query := s.db.WithContext(ctx).Where("is_active = ?", true)
	if learnerID > 0 {
		query = query.Where("(is_public = ? OR owner_learner_user_id = ?)", true, learnerID)
	} else {
		query = query.Where("is_public = ?", true)
	}
	if err := query.Find(&models).Error; err != nil {
		return storage.APIConfig{}, err
	}
	normalized := normalizeToolName(toolName)
	for _, model := range models {
		if normalizeToolName(resolvedAPIConfigToolName(model)) == normalized {
			return model, nil
		}
	}
	return storage.APIConfig{}, errors.New("api config tool does not exist")
}

func (s *Service) findAPIConfigForActor(ctx context.Context, id uint, learnerID *uint) (storage.APIConfig, error) {
	return s.findAPIConfigForActorTx(s.db.WithContext(ctx), id, learnerID)
}

func (s *Service) findAPIConfigForActorTx(tx *gorm.DB, id uint, learnerID *uint) (storage.APIConfig, error) {
	var model storage.APIConfig
	query := tx.Where("id = ?", id)
	if learnerID != nil {
		query = query.Where("owner_learner_user_id = ?", *learnerID)
	}
	if err := query.First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return storage.APIConfig{}, errors.New("api config does not exist")
		}
		return storage.APIConfig{}, err
	}
	return model, nil
}

func (s *Service) syncAPIConfigToolConfigTx(tx *gorm.DB, model storage.APIConfig, previousResolvedToolName string) error {
	resolvedToolName := resolvedAPIConfigToolName(model)
	if strings.TrimSpace(resolvedToolName) == "" {
		return errors.New("resolved api config tool name is required")
	}

	requiresMembership := false
	if previousResolvedToolName != "" {
		var previous storage.MCPToolConfig
		if err := tx.Where("tool_name = ?", previousResolvedToolName).First(&previous).Error; err == nil {
			requiresMembership = previous.RequiresMembership
			if previousResolvedToolName != resolvedToolName {
				if err := tx.Delete(&previous).Error; err != nil {
					return err
				}
			}
		}
	}

	var tool storage.MCPToolConfig
	err := tx.Where("tool_name = ?", resolvedToolName).First(&tool).Error
	switch {
	case err == nil:
		tool.Title = model.Name
		tool.Description = model.Description
		tool.Category = normalizeToolCategory(model.Category)
		tool.SourceType = apiConfigSourceType
		tool.IsEnabled = model.IsActive
		if tool.RequiresMembership != requiresMembership && previousResolvedToolName != "" && previousResolvedToolName != resolvedToolName {
			tool.RequiresMembership = requiresMembership
		}
		return tx.Save(&tool).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		tool = storage.MCPToolConfig{
			ToolName:           resolvedToolName,
			Title:              model.Name,
			Description:        model.Description,
			Category:           normalizeToolCategory(model.Category),
			SourceType:         apiConfigSourceType,
			IsEnabled:          model.IsActive,
			RequiresMembership: requiresMembership,
		}
		return tx.Create(&tool).Error
	default:
		return err
	}
}

func (s *Service) ensureToolNameIsAvailable(ctx context.Context, toolName string, excludeConfigID uint, previousResolvedToolName string) error {
	return s.ensureToolNameIsAvailableTx(s.db.WithContext(ctx), normalizeToolName(toolName), excludeConfigID, previousResolvedToolName)
}

func (s *Service) ensureToolNameIsAvailableTx(tx *gorm.DB, toolName string, excludeConfigID uint, previousResolvedToolName string) error {
	toolName = normalizeToolName(toolName)
	if toolName == "" {
		return errors.New("tool name is required")
	}
	if previousResolvedToolName != "" && toolName == normalizeToolName(previousResolvedToolName) {
		return nil
	}

	var existing storage.MCPToolConfig
	if err := tx.Where("tool_name = ?", toolName).First(&existing).Error; err == nil {
		return errors.New("tool name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if excludeConfigID > 0 {
		var configs []storage.APIConfig
		if err := tx.Where("id <> ?", excludeConfigID).Find(&configs).Error; err != nil {
			return err
		}
		for _, config := range configs {
			if normalizeToolName(resolvedAPIConfigToolName(config)) == toolName {
				return errors.New("tool name already exists")
			}
		}
	}
	return nil
}

func parseAPIParameterDefinitions(raw string) ([]apiParameterDefinition, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	if !strings.HasPrefix(raw, "{") && !strings.HasPrefix(raw, "[") {
		replacer := strings.NewReplacer("\r\n", "\n", "\r", "\n", ",", "\n")
		chunks := strings.Split(replacer.Replace(raw), "\n")
		result := make([]apiParameterDefinition, 0, len(chunks))
		for _, chunk := range chunks {
			name := strings.TrimSpace(chunk)
			if name == "" {
				continue
			}
			result = append(result, apiParameterDefinition{
				Name:        name,
				Type:        "string",
				Description: fmt.Sprintf("Parameter: %s", name),
				Required:    true,
			})
		}
		return result, nil
	}

	if strings.HasPrefix(raw, "[") {
		var list []interface{}
		if err := json.Unmarshal([]byte(raw), &list); err != nil {
			return nil, errors.New("parameters must be valid JSON")
		}
		result := make([]apiParameterDefinition, 0, len(list))
		for _, item := range list {
			switch typed := item.(type) {
			case string:
				name := strings.TrimSpace(typed)
				if name == "" {
					continue
				}
				result = append(result, apiParameterDefinition{
					Name:        name,
					Type:        "string",
					In:          "url",
					Description: fmt.Sprintf("Parameter: %s", name),
					Required:    true,
				})
			case map[string]interface{}:
				name := strings.TrimSpace(fmt.Sprintf("%v", typed["name"]))
				if name == "" {
					continue
				}
				description := strings.TrimSpace(fmt.Sprintf("%v", typed["description"]))
				required, _ := typed["required"].(bool)
				paramType := strings.TrimSpace(fmt.Sprintf("%v", typed["type"]))
				paramIn := strings.TrimSpace(fmt.Sprintf("%v", typed["in"]))
				if paramIn == "" {
					paramIn = strings.TrimSpace(fmt.Sprintf("%v", typed["location"]))
				}
				if description == "" {
					description = fmt.Sprintf("Parameter: %s", name)
				}
				result = append(result, apiParameterDefinition{
					Name:        name,
					Type:        paramType,
					In:          paramIn,
					Description: description,
					Required:    required,
				})
			}
		}
		return result, nil
	}

	var object map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &object); err != nil {
		return nil, errors.New("parameters must be valid JSON")
	}

	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]apiParameterDefinition, 0, len(keys))
	for _, key := range keys {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		value := object[key]
		switch typed := value.(type) {
		case string:
			result = append(result, apiParameterDefinition{
				Name:        name,
				Type:        "string",
				In:          "url",
				Description: strings.TrimSpace(typed),
			})
		case map[string]interface{}:
			description := strings.TrimSpace(fmt.Sprintf("%v", typed["description"]))
			required, _ := typed["required"].(bool)
			paramType := strings.TrimSpace(fmt.Sprintf("%v", typed["type"]))
			paramIn := strings.TrimSpace(fmt.Sprintf("%v", typed["in"]))
			if paramIn == "" {
				paramIn = strings.TrimSpace(fmt.Sprintf("%v", typed["location"]))
			}
			if description == "" {
				description = fmt.Sprintf("Parameter: %s", name)
			}
			result = append(result, apiParameterDefinition{
				Name:        name,
				Type:        paramType,
				In:          paramIn,
				Description: description,
				Required:    required,
			})
		default:
			result = append(result, apiParameterDefinition{
				Name:        name,
				Type:        "string",
				In:          "url",
				Description: fmt.Sprintf("Parameter: %s", name),
			})
		}
	}
	return result, nil
}

func validateAPIConfigInput(name, toolName, rawURL, method, headers, body, parameters string) (storage.APIConfig, error) {
	validated := storage.APIConfig{
		Name:     strings.TrimSpace(name),
		ToolName: normalizeToolName(strings.ReplaceAll(strings.TrimSpace(toolName), ".", "_")),
		URL:      strings.TrimSpace(rawURL),
		Method:   strings.ToUpper(strings.TrimSpace(method)),
	}
	if validated.Name == "" {
		return storage.APIConfig{}, errors.New("name is required")
	}
	if validated.URL == "" {
		return storage.APIConfig{}, errors.New("url is required")
	}
	if !strings.HasPrefix(validated.URL, "/") {
		parsed, err := url.Parse(validated.URL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return storage.APIConfig{}, errors.New("url must be an absolute http/https url or a relative path")
		}
		if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
			return storage.APIConfig{}, errors.New("url must use http or https")
		}
	}
	if validated.Method == "" {
		validated.Method = http.MethodGet
	}
	switch validated.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead:
	default:
		return storage.APIConfig{}, errors.New("http method must be GET, POST, PUT, PATCH, DELETE, or HEAD")
	}
	if validated.ToolName != "" && !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(validated.ToolName) {
		return storage.APIConfig{}, errors.New("tool name may only contain lowercase letters, numbers, underscores, and hyphens")
	}
	if err := validateJSONObjectField("headers", headers); err != nil {
		return storage.APIConfig{}, err
	}
	if err := validateJSONField("body", body); err != nil {
		return storage.APIConfig{}, err
	}
	definitions, err := parseAPIParameterDefinitions(parameters)
	if err != nil {
		return storage.APIConfig{}, err
	}
	for _, definition := range definitions {
		location := strings.ToLower(strings.TrimSpace(definition.In))
		if location == "header" && !isValidHTTPHeaderFieldName(strings.TrimSpace(definition.Name)) {
			return storage.APIConfig{}, fmt.Errorf("header parameter name %q is invalid", definition.Name)
		}
	}
	return validated, nil
}

func validateJSONObjectField(label, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var object map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &object); err != nil {
		return fmt.Errorf("%s must be a valid JSON object", label)
	}
	if strings.EqualFold(label, "headers") {
		for key := range object {
			if !isValidHTTPHeaderFieldName(strings.TrimSpace(key)) {
				return fmt.Errorf("headers contains an invalid header name: %s", key)
			}
		}
	}
	return nil
}

func validateJSONField(label, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("%s must be valid JSON", label)
	}
	return nil
}

func resolvedAPIConfigToolName(model storage.APIConfig) string {
	if custom := normalizeToolName(model.ToolName); custom != "" {
		return custom
	}
	category := normalizeKey(model.Category)
	if category == "" {
		category = "config"
	}
	return normalizeToolName(fmt.Sprintf("api_%s_%d", category, model.ID))
}

func normalizeAPIParameterType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "integer", "int":
		return "integer"
	case "number", "float":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "object":
		return "object"
	case "array":
		return "array"
	default:
		return "string"
	}
}

func extractAPITemplateKeys(values ...string) []string {
	seen := map[string]struct{}{}
	keys := make([]string, 0, 8)
	for _, value := range values {
		matches := apiTemplatePattern.FindAllStringSubmatch(value, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			key := strings.TrimSpace(match[1])
			if key == "" {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func resolveAPIConfigURL(rawURL string, execCtx domain.APIConfigExecutionContext, args map[string]interface{}) (string, error) {
	rendered := renderTemplateString(rawURL, buildAPIConfigTemplateValues(execCtx, args))
	rendered = strings.TrimSpace(rendered)
	if rendered == "" {
		return "", errors.New("resolved api config url is empty")
	}
	if strings.HasPrefix(rendered, "/") {
		base := strings.TrimRight(strings.TrimSpace(execCtx.HTTPBaseURL), "/")
		if base == "" {
			return "", errors.New("relative api config url requires current http base url")
		}
		return base + rendered, nil
	}
	return rendered, nil
}

func renderAPIConfigHeaders(raw string, execCtx domain.APIConfigExecutionContext, args map[string]interface{}) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}
	var headers map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return nil, errors.New("headers must be a valid JSON object")
	}
	values := buildAPIConfigTemplateValues(execCtx, args)
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		rendered := renderTemplateValue(value, values)
		result[key] = strings.TrimSpace(fmt.Sprintf("%v", rendered))
	}
	return result, nil
}

func renderAPIConfigBody(raw string, execCtx domain.APIConfigExecutionContext, args map[string]interface{}) ([]byte, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", nil
	}
	var payload interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, "", errors.New("body must be valid JSON")
	}
	values := buildAPIConfigTemplateValues(execCtx, args)
	rendered := renderTemplateValue(payload, values)
	data, err := json.Marshal(rendered)
	if err != nil {
		return nil, "", err
	}
	return data, "application/json", nil
}

func appendAPIConfigQueryArgs(
	resolvedURL string,
	model storage.APIConfig,
	execCtx domain.APIConfigExecutionContext,
	args map[string]interface{},
) (string, error) {
	parsed, err := url.Parse(resolvedURL)
	if err != nil {
		return "", err
	}

	queryValues := parsed.Query()
	templateValues := buildAPIConfigTemplateValues(execCtx, args)
	parameterDefs, _ := parseAPIParameterDefinitions(model.Parameters)
	if len(parameterDefs) == 0 {
		for key, value := range args {
			if value == nil {
				continue
			}
			if apiConfigQueryHasConcreteValue(queryValues, key) {
				continue
			}
			queryValues.Set(key, fmt.Sprintf("%v", renderTemplateValue(value, templateValues)))
		}
	} else {
		for _, definition := range parameterDefs {
			if location := strings.ToLower(strings.TrimSpace(definition.In)); location != "" && location != "url" && location != "query" {
				continue
			}
			value, exists := args[definition.Name]
			if !exists || value == nil {
				continue
			}
			if apiConfigQueryHasConcreteValue(queryValues, definition.Name) {
				continue
			}
			queryValues.Set(definition.Name, fmt.Sprintf("%v", renderTemplateValue(value, templateValues)))
		}
	}

	parsed.RawQuery = queryValues.Encode()
	return parsed.String(), nil
}

func apiConfigQueryHasConcreteValue(values url.Values, key string) bool {
	existing, exists := values[key]
	if !exists {
		return false
	}
	for _, value := range existing {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func isValidHTTPHeaderFieldName(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	const allowed = "!#$%&'*+-.^_`|~"
	for i := 0; i < len(value); i++ {
		char := value[i]
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case strings.ContainsRune(allowed, rune(char)):
		default:
			return false
		}
	}
	return true
}

func buildAPIConfigTemplateValues(execCtx domain.APIConfigExecutionContext, args map[string]interface{}) map[string]interface{} {
	values := map[string]interface{}{
		"subject_key":      execCtx.SubjectKey,
		"subject":          execCtx.SubjectKey,
		"access_token":     execCtx.AccessToken,
		"learner_user_id":  execCtx.LearnerUserID,
		"user_id":          execCtx.LearnerUserID,
		"learner_username": execCtx.Username,
		"username":         execCtx.Username,
	}
	for key, value := range args {
		values[key] = value
	}
	return values
}

func renderTemplateValue(value interface{}, values map[string]interface{}) interface{} {
	switch typed := value.(type) {
	case string:
		return renderTemplateStringAny(typed, values)
	case []interface{}:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, renderTemplateValue(item, values))
		}
		return items
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			result[key] = renderTemplateValue(item, values)
		}
		return result
	default:
		return value
	}
}

func renderTemplateString(value string, values map[string]interface{}) string {
	return fmt.Sprintf("%v", renderTemplateStringAny(value, values))
}

func renderTemplateStringAny(value string, values map[string]interface{}) interface{} {
	trimmed := strings.TrimSpace(value)
	if matches := apiTemplatePattern.FindStringSubmatch(trimmed); len(matches) == 2 && matches[0] == trimmed {
		if resolved, exists := values[matches[1]]; exists {
			return resolved
		}
		return ""
	}
	return apiTemplatePattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := apiTemplatePattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if resolved, exists := values[parts[1]]; exists {
			return fmt.Sprintf("%v", resolved)
		}
		return ""
	})
}
