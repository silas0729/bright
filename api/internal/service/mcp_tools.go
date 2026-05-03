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
	if len(definitions) == 0 {
		return nil
	}

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
		return nil
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
