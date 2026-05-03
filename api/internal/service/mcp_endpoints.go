package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

var endpointQueryParamPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func (s *Service) ListLearnerMCPEndpoints(ctx context.Context, learnerID uint) ([]domain.MCPEndpoint, error) {
	if learnerID == 0 {
		return nil, errors.New("learner id is required")
	}

	var models []storage.LearnerMCPEndpoint
	if err := s.db.WithContext(ctx).
		Where("learner_user_id = ?", learnerID).
		Order("updated_at desc, id desc").
		Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]domain.MCPEndpoint, 0, len(models))
	for _, model := range models {
		items = append(items, toMCPEndpoint(model))
	}
	return items, nil
}

func (s *Service) CreateLearnerMCPEndpoint(ctx context.Context, learnerID uint, input domain.CreateMCPEndpointInput) (domain.MCPEndpoint, error) {
	if learnerID == 0 {
		return domain.MCPEndpoint{}, errors.New("learner id is required")
	}

	model, err := newLearnerMCPEndpointModel(learnerID, input.Name, input.URL, input.Description, input.Enabled, input.TokenQueryParam, input.SubjectQueryParam)
	if err != nil {
		return domain.MCPEndpoint{}, err
	}

	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.MCPEndpoint{}, err
	}
	return toMCPEndpoint(model), nil
}

func (s *Service) UpdateLearnerMCPEndpoint(ctx context.Context, learnerID uint, endpointID uint, input domain.UpdateMCPEndpointInput) (domain.MCPEndpoint, error) {
	if learnerID == 0 {
		return domain.MCPEndpoint{}, errors.New("learner id is required")
	}
	if endpointID == 0 {
		return domain.MCPEndpoint{}, errors.New("endpoint id is required")
	}

	var model storage.LearnerMCPEndpoint
	if err := s.db.WithContext(ctx).
		Where("id = ? AND learner_user_id = ?", endpointID, learnerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.MCPEndpoint{}, errors.New("endpoint does not exist")
		}
		return domain.MCPEndpoint{}, err
	}

	normalized, err := newLearnerMCPEndpointModel(learnerID, input.Name, input.URL, input.Description, input.Enabled, input.TokenQueryParam, input.SubjectQueryParam)
	if err != nil {
		return domain.MCPEndpoint{}, err
	}

	updates := map[string]any{
		"name":                normalized.Name,
		"url":                 normalized.URL,
		"description":         normalized.Description,
		"enabled":             normalized.Enabled,
		"token_query_param":   normalized.TokenQueryParam,
		"subject_query_param": normalized.SubjectQueryParam,
	}
	if err := s.db.WithContext(ctx).Model(&model).Updates(updates).Error; err != nil {
		return domain.MCPEndpoint{}, err
	}
	if err := s.db.WithContext(ctx).
		Where("id = ? AND learner_user_id = ?", endpointID, learnerID).
		First(&model).Error; err != nil {
		return domain.MCPEndpoint{}, err
	}

	return toMCPEndpoint(model), nil
}

func (s *Service) DeleteLearnerMCPEndpoint(ctx context.Context, learnerID uint, endpointID uint) error {
	if learnerID == 0 {
		return errors.New("learner id is required")
	}
	if endpointID == 0 {
		return errors.New("endpoint id is required")
	}

	result := s.db.WithContext(ctx).
		Where("id = ? AND learner_user_id = ?", endpointID, learnerID).
		Delete(&storage.LearnerMCPEndpoint{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("endpoint does not exist")
	}
	return nil
}

func (s *Service) GetLearnerMCPEndpoint(ctx context.Context, learnerID uint, endpointID uint) (domain.MCPEndpoint, error) {
	if learnerID == 0 {
		return domain.MCPEndpoint{}, errors.New("learner id is required")
	}
	if endpointID == 0 {
		return domain.MCPEndpoint{}, errors.New("endpoint id is required")
	}

	var model storage.LearnerMCPEndpoint
	if err := s.db.WithContext(ctx).
		Where("id = ? AND learner_user_id = ?", endpointID, learnerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.MCPEndpoint{}, errors.New("endpoint does not exist")
		}
		return domain.MCPEndpoint{}, err
	}
	return toMCPEndpoint(model), nil
}

func (s *Service) ListEnabledLearnerMCPEndpoints(ctx context.Context, learnerID uint) ([]domain.MCPEndpoint, error) {
	if learnerID == 0 {
		return nil, errors.New("learner id is required")
	}

	var models []storage.LearnerMCPEndpoint
	if err := s.db.WithContext(ctx).
		Where("learner_user_id = ? AND enabled = ?", learnerID, true).
		Order("updated_at desc, id desc").
		Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]domain.MCPEndpoint, 0, len(models))
	for _, model := range models {
		items = append(items, toMCPEndpoint(model))
	}
	return items, nil
}

func (s *Service) ListAllEnabledLearnerMCPEndpoints(ctx context.Context) ([]domain.MCPEndpoint, error) {
	var models []storage.LearnerMCPEndpoint
	if err := s.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("updated_at desc, id desc").
		Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]domain.MCPEndpoint, 0, len(models))
	for _, model := range models {
		items = append(items, toMCPEndpoint(model))
	}
	return items, nil
}

func newLearnerMCPEndpointModel(
	learnerID uint,
	name string,
	rawURL string,
	description string,
	enabled bool,
	tokenQueryParam string,
	subjectQueryParam string,
) (storage.LearnerMCPEndpoint, error) {
	normalizedURL, err := normalizeMCPEndpointURL(rawURL)
	if err != nil {
		return storage.LearnerMCPEndpoint{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		parsed, parseErr := url.Parse(normalizedURL)
		if parseErr == nil && parsed.Host != "" {
			name = parsed.Host
		}
	}
	if name == "" {
		return storage.LearnerMCPEndpoint{}, errors.New("endpoint name is required")
	}

	tokenQueryParam, err = normalizeMCPEndpointQueryParam(tokenQueryParam, "token_query_param")
	if err != nil {
		return storage.LearnerMCPEndpoint{}, err
	}
	subjectQueryParam, err = normalizeMCPEndpointQueryParam(subjectQueryParam, "subject_query_param")
	if err != nil {
		return storage.LearnerMCPEndpoint{}, err
	}

	return storage.LearnerMCPEndpoint{
		LearnerUserID:     learnerID,
		Name:              name,
		URL:               normalizedURL,
		Description:       strings.TrimSpace(description),
		Enabled:           enabled,
		TokenQueryParam:   tokenQueryParam,
		SubjectQueryParam: subjectQueryParam,
	}, nil
}

func normalizeMCPEndpointURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("endpoint url is required")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("endpoint url is invalid")
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return "", errors.New("endpoint url must start with ws:// or wss://")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", errors.New("endpoint url host is required")
	}

	parsed.Fragment = ""
	return parsed.String(), nil
}

func normalizeMCPEndpointQueryParam(value string, fieldName string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !endpointQueryParamPattern.MatchString(value) {
		return "", errors.New(fieldName + " must contain only letters, numbers, dot, underscore, or hyphen")
	}
	return value, nil
}

func toMCPEndpoint(model storage.LearnerMCPEndpoint) domain.MCPEndpoint {
	return domain.MCPEndpoint{
		ID:                model.ID,
		LearnerUserID:     model.LearnerUserID,
		Name:              model.Name,
		URL:               model.URL,
		Description:       model.Description,
		Enabled:           model.Enabled,
		TokenQueryParam:   model.TokenQueryParam,
		SubjectQueryParam: model.SubjectQueryParam,
		ConnectionStatus:  strings.TrimSpace(model.ConnectionStatus),
		IsConnected:       strings.EqualFold(strings.TrimSpace(model.ConnectionStatus), "connected"),
		LastError:         strings.TrimSpace(model.LastError),
		ConnectedAt:       model.ConnectedAt,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

func (s *Service) UpdateLearnerMCPEndpointConnectionState(
	ctx context.Context,
	learnerID uint,
	endpointID uint,
	status string,
	lastError string,
	connectedAt *time.Time,
) error {
	if learnerID == 0 {
		return errors.New("learner id is required")
	}
	if endpointID == 0 {
		return errors.New("endpoint id is required")
	}

	normalizedStatus := strings.TrimSpace(strings.ToLower(status))
	switch normalizedStatus {
	case "", "disconnected", "connecting", "connected", "error":
	default:
		return errors.New("invalid connection status")
	}
	if normalizedStatus == "" {
		normalizedStatus = "disconnected"
	}

	updates := map[string]any{
		"connection_status": normalizedStatus,
		"last_error":        strings.TrimSpace(lastError),
		"connected_at":      connectedAt,
	}
	return s.db.WithContext(ctx).
		Model(&storage.LearnerMCPEndpoint{}).
		Where("id = ? AND learner_user_id = ?", endpointID, learnerID).
		Updates(updates).Error
}
