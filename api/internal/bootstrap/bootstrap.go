package bootstrap

import (
	"context"
	"errors"
	"strings"

	"brights/api/internal/domain"
	"brights/api/internal/service"
)

func SuperAdmin(ctx context.Context, svc *service.Service, username, password, displayName string) (domain.AdminUser, bool, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" {
		return domain.AdminUser{}, false, errors.New("bootstrap username is required")
	}
	if password == "" {
		return domain.AdminUser{}, false, errors.New("bootstrap password is required")
	}
	return svc.BootstrapSuperAdmin(ctx, domain.BootstrapAdminInput{
		Username:    username,
		Password:    password,
		DisplayName: displayName,
	})
}

func ResetSuperAdminPassword(ctx context.Context, svc *service.Service, username, password, displayName string) (domain.AdminUser, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return domain.AdminUser{}, errors.New("new password is required")
	}
	return svc.ResetSuperAdminPassword(ctx, username, password, displayName)
}
