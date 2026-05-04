package service

import (
	"context"
	"errors"
	"sort"
	"strings"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
	"gorm.io/gorm"
)

func (s *Service) SyncDefaultMCPToolConfigs(ctx context.Context) error {
	definitions := domain.DefaultMCPToolDefinitions()
	return s.db.WithContext(ctx).Transaction(func(tx *gormDBShim) error {
		for _, definition := range definitions {
			toolName := normalizeToolName(definition.Name)
			if toolName == "" {
				continue
			}

			var model storage.MCPToolConfig
			err := tx.Where("tool_name = ?", toolName).First(&model).Error
			switch {
			case err == nil:
				updates := map[string]any{
					"title":       strings.TrimSpace(definition.Title),
					"description": strings.TrimSpace(definition.Description),
					"category":    normalizeToolCategory(definition.Category),
					"source_type": normalizeToolSourceType(definition.SourceType),
				}
				if err := tx.Model(&model).Updates(updates).Error; err != nil {
					return err
				}
			case errors.Is(err, recordNotFoundShim):
				model = storage.MCPToolConfig{
					ToolName:           toolName,
					Title:              strings.TrimSpace(definition.Title),
					Description:        strings.TrimSpace(definition.Description),
					Category:           normalizeToolCategory(definition.Category),
					SourceType:         normalizeToolSourceType(definition.SourceType),
					IsEnabled:          definition.DefaultEnabled,
					RequiresMembership: definition.DefaultRequiresMembership,
				}
				if err := tx.Create(&model).Error; err != nil {
					return err
				}
			default:
				return err
			}
		}
		return s.syncAPIConfigMCPToolConfigsTx(tx)
	})
}

func (s *Service) ListMCPToolConfigs(ctx context.Context) ([]domain.MCPToolConfig, error) {
	if err := s.SyncDefaultMCPToolConfigs(ctx); err != nil {
		return nil, err
	}

	var models []storage.MCPToolConfig
	if err := s.db.WithContext(ctx).
		Order("category asc, tool_name asc").
		Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]domain.MCPToolConfig, 0, len(models))
	for _, model := range models {
		items = append(items, toMCPToolConfig(model))
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Category == items[j].Category {
			return items[i].ToolName < items[j].ToolName
		}
		return items[i].Category < items[j].Category
	})

	return items, nil
}

func (s *Service) ListAdminMCPToolConfigs(ctx context.Context, filter domain.MCPToolConfigFilter) (domain.PagedMCPToolConfigs, error) {
	if err := s.SyncDefaultMCPToolConfigs(ctx); err != nil {
		return domain.PagedMCPToolConfigs{}, err
	}

	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)

	query := s.db.WithContext(ctx).Model(&storage.MCPToolConfig{})
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"tool_name LIKE ? OR title LIKE ? OR description LIKE ? OR category LIKE ? OR source_type LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		query = query.Where("category = ?", normalizeToolCategory(category))
	}
	if filter.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *filter.IsEnabled)
	}
	if filter.RequiresMembership != nil {
		query = query.Where("requires_membership = ?", *filter.RequiresMembership)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedMCPToolConfigs{}, err
	}

	var models []storage.MCPToolConfig
	if err := query.Order("category asc, tool_name asc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedMCPToolConfigs{}, err
	}

	items := make([]domain.MCPToolConfig, 0, len(models))
	for _, model := range models {
		items = append(items, toMCPToolConfig(model))
	}

	return domain.PagedMCPToolConfigs{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) GetMCPToolConfigMap(ctx context.Context) (map[string]domain.MCPToolConfig, error) {
	items, err := s.ListMCPToolConfigs(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]domain.MCPToolConfig, len(items))
	for _, item := range items {
		result[item.ToolName] = item
	}
	return result, nil
}

func (s *Service) UpdateMCPToolConfig(ctx context.Context, toolName string, input domain.UpdateMCPToolConfigInput) (domain.MCPToolConfig, error) {
	if err := s.SyncDefaultMCPToolConfigs(ctx); err != nil {
		return domain.MCPToolConfig{}, err
	}

	toolName = normalizeToolName(toolName)
	if toolName == "" {
		return domain.MCPToolConfig{}, errors.New("tool name is required")
	}

	var model storage.MCPToolConfig
	if err := s.db.WithContext(ctx).Where("tool_name = ?", toolName).First(&model).Error; err != nil {
		if errors.Is(err, recordNotFoundShim) {
			return domain.MCPToolConfig{}, errors.New("tool does not exist")
		}
		return domain.MCPToolConfig{}, err
	}

	updates := map[string]any{}
	if input.IsEnabled != nil {
		updates["is_enabled"] = *input.IsEnabled
	}
	if input.RequiresMembership != nil {
		updates["requires_membership"] = *input.RequiresMembership
	}
	if len(updates) == 0 {
		return toMCPToolConfig(model), nil
	}

	if err := s.db.WithContext(ctx).Model(&model).Updates(updates).Error; err != nil {
		return domain.MCPToolConfig{}, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", model.ID).First(&model).Error; err != nil {
		return domain.MCPToolConfig{}, err
	}
	return toMCPToolConfig(model), nil
}

func toMCPToolConfig(model storage.MCPToolConfig) domain.MCPToolConfig {
	return domain.MCPToolConfig{
		ID:                 model.ID,
		ToolName:           model.ToolName,
		Title:              model.Title,
		Description:        model.Description,
		Category:           model.Category,
		SourceType:         model.SourceType,
		IsEnabled:          model.IsEnabled,
		RequiresMembership: model.RequiresMembership,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func (s *Service) syncAPIConfigMCPToolConfigsTx(tx *gormDBShim) error {
	var configs []storage.APIConfig
	if err := tx.Find(&configs).Error; err != nil {
		return err
	}

	activeNames := make(map[string]struct{}, len(configs))
	for _, config := range configs {
		resolvedToolName := resolvedAPIConfigToolName(config)
		if strings.TrimSpace(resolvedToolName) == "" {
			continue
		}
		activeNames[resolvedToolName] = struct{}{}

		var model storage.MCPToolConfig
		err := tx.Where("tool_name = ?", resolvedToolName).First(&model).Error
		switch {
		case err == nil:
			model.Title = strings.TrimSpace(config.Name)
			model.Description = strings.TrimSpace(config.Description)
			model.Category = normalizeToolCategory(config.Category)
			model.SourceType = apiConfigSourceType
			if !config.IsActive {
				model.IsEnabled = false
			}
			if err := tx.Save(&model).Error; err != nil {
				return err
			}
		case errors.Is(err, recordNotFoundShim):
			model = storage.MCPToolConfig{
				ToolName:           resolvedToolName,
				Title:              strings.TrimSpace(config.Name),
				Description:        strings.TrimSpace(config.Description),
				Category:           normalizeToolCategory(config.Category),
				SourceType:         apiConfigSourceType,
				IsEnabled:          config.IsActive,
				RequiresMembership: false,
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		default:
			return err
		}
	}

	var orphaned []storage.MCPToolConfig
	if err := tx.Where("source_type = ?", apiConfigSourceType).Find(&orphaned).Error; err != nil {
		return err
	}
	for _, item := range orphaned {
		if _, exists := activeNames[item.ToolName]; exists {
			continue
		}
		if err := tx.Delete(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeToolName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeToolCategory(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "general"
	}
	return value
}

func normalizeToolSourceType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "builtin"
	}
	return value
}

// These aliases keep this file focused while reusing gorm symbols without
// adding them to every call site in the patch body.
type gormDBShim = gorm.DB

var recordNotFoundShim = gorm.ErrRecordNotFound
