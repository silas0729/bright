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

func (s *Service) RegisterLearner(ctx context.Context, input domain.LearnerRegisterInput) (domain.LearnerUser, error) {
	username := normalizeKey(input.Username)
	password := strings.TrimSpace(input.Password)
	displayName := strings.TrimSpace(input.DisplayName)
	inviteCode := normalizeInviteCode(input.InviteCode)

	if username == "" {
		return domain.LearnerUser{}, errors.New("username is required")
	}
	if password == "" {
		return domain.LearnerUser{}, errors.New("password is required")
	}
	if len(password) < 8 {
		return domain.LearnerUser{}, errors.New("password must be at least 8 characters")
	}
	if displayName == "" {
		displayName = username
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&storage.LearnerUser{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return domain.LearnerUser{}, err
	}
	if count > 0 {
		return domain.LearnerUser{}, errors.New("username already exists")
	}

	var invitedByUserID *uint
	if inviteCode != "" {
		var inviter storage.LearnerUser
		if err := s.db.WithContext(ctx).Where("invite_code = ?", inviteCode).First(&inviter).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.LearnerUser{}, errors.New("invite code does not exist")
			}
			return domain.LearnerUser{}, err
		}
		invitedByUserID = uintPtr(inviter.ID)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.LearnerUser{}, err
	}

	model := storage.LearnerUser{
		Username:        username,
		PasswordHash:    string(hash),
		DisplayName:     displayName,
		Status:          "active",
		InvitedByUserID: invitedByUserID,
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.LearnerUser{}, err
	}
	if err := s.ensureInviteCode(ctx, &model); err != nil {
		return domain.LearnerUser{}, err
	}
	return toLearnerUser(model), nil
}

func (s *Service) AuthenticateLearner(ctx context.Context, username, password string) (domain.LearnerUser, error) {
	username = normalizeKey(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return domain.LearnerUser{}, errors.New("username and password are required")
	}

	var model storage.LearnerUser
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.LearnerUser{}, errors.New("invalid username or password")
		}
		return domain.LearnerUser{}, err
	}
	if model.Status != "active" {
		return domain.LearnerUser{}, errors.New("user account is disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(model.PasswordHash), []byte(password)); err != nil {
		return domain.LearnerUser{}, errors.New("invalid username or password")
	}
	return toLearnerUser(model), nil
}

func (s *Service) GetLearnerByID(ctx context.Context, id uint) (domain.LearnerUser, error) {
	if id == 0 {
		return domain.LearnerUser{}, errors.New("user id is required")
	}

	var model storage.LearnerUser
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.LearnerUser{}, errors.New("user does not exist")
		}
		return domain.LearnerUser{}, err
	}
	return toLearnerUser(model), nil
}

func (s *Service) GetLearnerByIDWithMembership(ctx context.Context, id uint, subjectKey string) (domain.LearnerUser, error) {
	user, err := s.GetLearnerByID(ctx, id)
	if err != nil {
		return domain.LearnerUser{}, err
	}

	subjectKey = normalizeKey(subjectKey)
	if subjectKey == "" {
		return user, nil
	}

	subscription, err := s.findLatestSubscriptionStatus(ctx, user.Username, subjectKey)
	if err != nil {
		return domain.LearnerUser{}, err
	}
	user.Membership = subscription
	return user, nil
}

func toLearnerUser(model storage.LearnerUser) domain.LearnerUser {
	return domain.LearnerUser{
		ID:          model.ID,
		Username:    model.Username,
		DisplayName: model.DisplayName,
		Status:      model.Status,
		InviteCode:  model.InviteCode,
		CreatedAt:   model.CreatedAt,
	}
}
