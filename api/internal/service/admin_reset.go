package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func (s *Service) ResetSuperAdminPassword(ctx context.Context, username, newPassword, displayName string) (domain.AdminUser, error) {
	newPassword = strings.TrimSpace(newPassword)
	displayName = strings.TrimSpace(displayName)
	if newPassword == "" {
		return domain.AdminUser{}, errors.New("password is required")
	}
	if len(newPassword) < 8 {
		return domain.AdminUser{}, errors.New("password must be at least 8 characters")
	}

	model, err := s.resolveSuperAdminForReset(ctx, username)
	if err != nil {
		return domain.AdminUser{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return domain.AdminUser{}, err
	}

	updates := map[string]any{
		"password_hash": string(hash),
		"status":        "active",
		"role":          "super_admin",
		"is_super":      true,
	}
	if displayName != "" {
		updates["display_name"] = displayName
	}

	if err := s.db.WithContext(ctx).Model(&model).Updates(updates).Error; err != nil {
		return domain.AdminUser{}, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", model.ID).First(&model).Error; err != nil {
		return domain.AdminUser{}, err
	}
	return toAdminUser(model), nil
}

func (s *Service) resolveSuperAdminForReset(ctx context.Context, username string) (storage.AdminUser, error) {
	username = normalizeKey(username)
	if username != "" {
		var model storage.AdminUser
		if err := s.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.AdminUser{}, errors.New("super admin does not exist")
			}
			return storage.AdminUser{}, err
		}
		if !model.IsSuper && model.Role != "super_admin" {
			return storage.AdminUser{}, errors.New("target account is not a super admin")
		}
		return model, nil
	}

	var models []storage.AdminUser
	if err := s.db.WithContext(ctx).
		Where("is_super = ? OR role = ?", true, "super_admin").
		Order("id asc").
		Find(&models).Error; err != nil {
		return storage.AdminUser{}, err
	}
	if len(models) == 0 {
		return storage.AdminUser{}, errors.New("super admin does not exist")
	}
	if len(models) > 1 {
		return storage.AdminUser{}, errors.New("multiple super admins found, please specify username")
	}
	return models[0], nil
}
