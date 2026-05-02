package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func (s *Service) GetAdminSetupStatus(ctx context.Context) (domain.AdminSetupStatus, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&storage.AdminUser{}).Count(&count).Error; err != nil {
		return domain.AdminSetupStatus{}, err
	}
	return domain.AdminSetupStatus{
		Initialized: count > 0,
		AdminCount:  count,
	}, nil
}

func (s *Service) InitializeSuperAdmin(ctx context.Context, input domain.InitializeSuperAdminInput) (domain.AdminUser, error) {
	status, err := s.GetAdminSetupStatus(ctx)
	if err != nil {
		return domain.AdminUser{}, err
	}
	if status.Initialized {
		return domain.AdminUser{}, errors.New("admin has already been initialized")
	}

	username := normalizeKey(input.Username)
	password := strings.TrimSpace(input.Password)
	displayName := strings.TrimSpace(input.DisplayName)
	if username == "" {
		return domain.AdminUser{}, errors.New("username is required")
	}
	if password == "" {
		return domain.AdminUser{}, errors.New("password is required")
	}
	if len(password) < 8 {
		return domain.AdminUser{}, errors.New("password must be at least 8 characters")
	}
	if displayName == "" {
		displayName = "超级管理员"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AdminUser{}, err
	}

	model := storage.AdminUser{
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Role:         "super_admin",
		IsSuper:      true,
		Status:       "active",
	}

	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.AdminUser{}, err
	}
	return toAdminUser(model), nil
}
