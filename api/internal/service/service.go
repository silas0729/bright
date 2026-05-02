package service

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"brights/api/internal/catalog"
	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

const topicCategoryKind = "topic"

type Service struct {
	db        *gorm.DB
	captchaMu sync.Mutex
	captchas  map[string]captchaEntry
}

func New(db *gorm.DB) *Service {
	return &Service{
		db:       db,
		captchas: make(map[string]captchaEntry),
	}
}

func (s *Service) SeedDefaults(ctx context.Context) error {
	defaultSubject := storage.Subject{
		Key:         "english",
		Name:        "英语高频词汇",
		Description: "围绕真实场景整理的高频英语单词与学习内容。",
		Sort:        1,
		Featured:    true,
	}

	if err := s.upsertSubject(ctx, defaultSubject); err != nil {
		return err
	}

	defaultPlans := []storage.Plan{
		{
			Key:             "starter-monthly",
			Name:            "月度会员",
			BillingMode:     "monthly",
			PriceCents:      2900,
			Description:     "适合持续学习，按月开通，后续也方便升级为自动续费方案。",
			Recommended:     true,
			PaymentChannels: storage.JSONStringSlice{"wechat_jsapi", "wechat_native", "wechat_contract_pay"},
			Features:        storage.JSONStringSlice{"高频单词词库", "场景分类学习", "收藏与复习记录", "持续更新内容"},
		},
		{
			Key:             "lifetime",
			Name:            "终身买断",
			BillingMode:     "lifetime",
			PriceCents:      29900,
			Description:     "一次购买，长期可用，适合希望稳定积累学习内容的用户。",
			Recommended:     false,
			PaymentChannels: storage.JSONStringSlice{"wechat_jsapi", "wechat_native"},
			Features:        storage.JSONStringSlice{"长期可用", "后续专题内容持续开放", "适合长期学习规划"},
		},
	}

	for _, plan := range defaultPlans {
		if err := s.upsertPlan(ctx, plan); err != nil {
			return err
		}
	}

	defaultRoles := []storage.AdminRole{
		{
			Key:         "super_admin",
			Name:        "超级管理员",
			Description: "拥有后台全部管理权限。",
			Permissions: storage.JSONStringSlice{"*"},
			System:      true,
			Sort:        1,
		},
		{
			Key:         "content_admin",
			Name:        "内容管理员",
			Description: "负责学科、词库、分类、年级与支付配置管理。",
			Permissions: storage.JSONStringSlice{"admin.read", "subject.read", "subject.write", "catalog.read", "catalog.write", "grade.read", "grade.write", "plan.read", "plan.write", "payment.read", "payment.write", "site.read", "site.write", "learner.read", "learner.write"},
			System:      true,
			Sort:        2,
		},
		{
			Key:         "viewer",
			Name:        "只读查看",
			Description: "用于运营查看数据，不可修改内容。",
			Permissions: storage.JSONStringSlice{"subject.read", "catalog.read", "grade.read", "admin.read", "plan.read", "payment.read", "site.read", "learner.read"},
			System:      true,
			Sort:        3,
		},
	}

	for _, role := range defaultRoles {
		if err := s.upsertAdminRole(ctx, role); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) BootstrapSuperAdmin(ctx context.Context, input domain.BootstrapAdminInput) (domain.AdminUser, bool, error) {
	username := normalizeKey(input.Username)
	if username == "" {
		return domain.AdminUser{}, false, errors.New("username is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return domain.AdminUser{}, false, errors.New("password is required")
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = "超级管理员"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AdminUser{}, false, err
	}

	var model storage.AdminUser
	err = s.db.WithContext(ctx).Where("username = ?", username).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		model = storage.AdminUser{
			Username:     username,
			PasswordHash: string(passwordHash),
			DisplayName:  displayName,
			Role:         "super_admin",
			IsSuper:      true,
			Status:       "active",
		}
		if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
			return domain.AdminUser{}, false, err
		}
		return toAdminUser(model), true, nil
	}
	if err != nil {
		return domain.AdminUser{}, false, err
	}

	updates := map[string]any{
		"password_hash": string(passwordHash),
		"display_name":  displayName,
		"role":          "super_admin",
		"is_super":      true,
		"status":        "active",
	}
	if err := s.db.WithContext(ctx).Model(&model).Updates(updates).Error; err != nil {
		return domain.AdminUser{}, false, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", model.ID).First(&model).Error; err != nil {
		return domain.AdminUser{}, false, err
	}
	return toAdminUser(model), false, nil
}

func (s *Service) AuthenticateAdmin(ctx context.Context, username, password string) (domain.AdminUser, error) {
	username = normalizeKey(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return domain.AdminUser{}, errors.New("username and password are required")
	}

	var model storage.AdminUser
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AdminUser{}, errors.New("invalid username or password")
		}
		return domain.AdminUser{}, err
	}
	if model.Status != "active" {
		return domain.AdminUser{}, errors.New("admin account is disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(model.PasswordHash), []byte(password)); err != nil {
		return domain.AdminUser{}, errors.New("invalid username or password")
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&model).Update("last_login_at", &now).Error; err != nil {
		return domain.AdminUser{}, err
	}
	model.LastLoginAt = &now
	return toAdminUser(model), nil
}

func (s *Service) GetAdminByID(ctx context.Context, id uint) (domain.AdminUser, error) {
	if id == 0 {
		return domain.AdminUser{}, errors.New("admin id is required")
	}
	var model storage.AdminUser
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AdminUser{}, errors.New("admin does not exist")
		}
		return domain.AdminUser{}, err
	}
	return toAdminUser(model), nil
}

func (s *Service) ChangeAdminPassword(ctx context.Context, adminID uint, oldPassword, newPassword string) error {
	oldPassword = strings.TrimSpace(oldPassword)
	newPassword = strings.TrimSpace(newPassword)
	if adminID == 0 {
		return errors.New("admin id is required")
	}
	if oldPassword == "" || newPassword == "" {
		return errors.New("old_password and new_password are required")
	}
	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	var model storage.AdminUser
	if err := s.db.WithContext(ctx).Where("id = ?", adminID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("admin does not exist")
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(model.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("old password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&model).Update("password_hash", string(hash)).Error
}

func (s *Service) CreateAdminUser(ctx context.Context, input domain.CreateAdminUserInput) (domain.AdminUser, error) {
	username := normalizeKey(input.Username)
	password := strings.TrimSpace(input.Password)
	displayName := strings.TrimSpace(input.DisplayName)
	roleKey := normalizeRoleKey(input.Role)
	status, err := normalizeAdminStatus(input.Status)
	if err != nil {
		return domain.AdminUser{}, err
	}

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
		displayName = username
	}
	if roleKey == "" {
		roleKey = "content_admin"
	}

	isSuper := boolOrDefault(input.IsSuper, false)
	if isSuper {
		roleKey = "super_admin"
	}
	if roleKey == "super_admin" {
		isSuper = true
	}
	if err := s.ensureAdminRoleExists(ctx, roleKey); err != nil {
		return domain.AdminUser{}, err
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&storage.AdminUser{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return domain.AdminUser{}, err
	}
	if count > 0 {
		return domain.AdminUser{}, errors.New("admin username already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AdminUser{}, err
	}

	model := storage.AdminUser{
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Role:         roleKey,
		IsSuper:      isSuper,
		Status:       status,
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.AdminUser{}, err
	}
	return toAdminUser(model), nil
}

func (s *Service) UpdateAdminUser(ctx context.Context, adminID uint, operatorID uint, input domain.UpdateAdminUserInput) (domain.AdminUser, error) {
	if adminID == 0 {
		return domain.AdminUser{}, errors.New("admin id is required")
	}

	var model storage.AdminUser
	if err := s.db.WithContext(ctx).Where("id = ?", adminID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AdminUser{}, errors.New("admin does not exist")
		}
		return domain.AdminUser{}, err
	}

	nextDisplayName := strings.TrimSpace(input.DisplayName)
	if nextDisplayName == "" {
		nextDisplayName = model.DisplayName
	}

	nextRole := model.Role
	if trimmedRole := normalizeRoleKey(input.Role); trimmedRole != "" {
		nextRole = trimmedRole
	}

	nextIsSuper := model.IsSuper
	if input.IsSuper != nil {
		nextIsSuper = *input.IsSuper
	}
	if nextRole == "super_admin" {
		nextIsSuper = true
	}
	if nextIsSuper && nextRole != "super_admin" {
		return domain.AdminUser{}, errors.New("super admin account must use super_admin role")
	}
	if err := s.ensureAdminRoleExists(ctx, nextRole); err != nil {
		return domain.AdminUser{}, err
	}

	nextStatus := model.Status
	if strings.TrimSpace(input.Status) != "" {
		status, err := normalizeAdminStatus(input.Status)
		if err != nil {
			return domain.AdminUser{}, err
		}
		nextStatus = status
	}

	if adminID == operatorID && nextStatus != "active" {
		return domain.AdminUser{}, errors.New("cannot disable your own admin account")
	}

	if model.IsSuper && (!nextIsSuper || nextStatus != "active") {
		count, err := s.countActiveSuperAdmins(ctx, adminID)
		if err != nil {
			return domain.AdminUser{}, err
		}
		if count == 0 {
			return domain.AdminUser{}, errors.New("cannot disable or demote the last active super admin")
		}
	}

	updates := map[string]any{
		"display_name": nextDisplayName,
		"role":         nextRole,
		"is_super":     nextIsSuper,
		"status":       nextStatus,
	}

	if password := strings.TrimSpace(input.Password); password != "" {
		if len(password) < 8 {
			return domain.AdminUser{}, errors.New("password must be at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return domain.AdminUser{}, err
		}
		updates["password_hash"] = string(hash)
	}

	if err := s.db.WithContext(ctx).Model(&model).Updates(updates).Error; err != nil {
		return domain.AdminUser{}, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", adminID).First(&model).Error; err != nil {
		return domain.AdminUser{}, err
	}
	return toAdminUser(model), nil
}

func (s *Service) ListAdminUsers(ctx context.Context, filter domain.AdminUserFilter) (domain.PagedAdminUsers, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).Model(&storage.AdminUser{})

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("username LIKE ? OR display_name LIKE ?", like, like)
	}
	if role := strings.TrimSpace(filter.Role); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedAdminUsers{}, err
	}

	var models []storage.AdminUser
	if err := query.Order("id asc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedAdminUsers{}, err
	}

	items := make([]domain.AdminUser, 0, len(models))
	for _, model := range models {
		items = append(items, toAdminUser(model))
	}

	return domain.PagedAdminUsers{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ListAdminRoles(ctx context.Context) ([]domain.AdminRole, error) {
	var models []storage.AdminRole
	if err := s.db.WithContext(ctx).Order("sort asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.AdminRole, 0, len(models))
	for _, model := range models {
		items = append(items, toAdminRole(model))
	}
	return items, nil
}

func (s *Service) CreateAdminRole(ctx context.Context, input domain.CreateAdminRoleInput) (domain.AdminRole, error) {
	key := normalizeRoleKey(input.Key)
	name := strings.TrimSpace(input.Name)
	if key == "" {
		return domain.AdminRole{}, errors.New("role key is required")
	}
	if name == "" {
		return domain.AdminRole{}, errors.New("role name is required")
	}

	permissions := cleanPermissionSet(input.Permissions)
	if len(permissions) == 0 {
		return domain.AdminRole{}, errors.New("at least one permission is required")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&storage.AdminRole{}).Where("role_key = ?", key).Count(&count).Error; err != nil {
		return domain.AdminRole{}, err
	}
	if count > 0 {
		return domain.AdminRole{}, errors.New("role key already exists")
	}

	model := storage.AdminRole{
		Key:         key,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Permissions: storage.JSONStringSlice(permissions),
		System:      false,
		Sort:        input.Sort,
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.AdminRole{}, err
	}
	return toAdminRole(model), nil
}

func (s *Service) UpdateAdminRole(ctx context.Context, roleID uint, input domain.UpdateAdminRoleInput) (domain.AdminRole, error) {
	if roleID == 0 {
		return domain.AdminRole{}, errors.New("role id is required")
	}

	var model storage.AdminRole
	if err := s.db.WithContext(ctx).Where("id = ?", roleID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AdminRole{}, errors.New("role does not exist")
		}
		return domain.AdminRole{}, err
	}
	if model.System {
		return domain.AdminRole{}, errors.New("system roles cannot be edited from the admin UI")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.AdminRole{}, errors.New("role name is required")
	}

	permissions := cleanPermissionSet(input.Permissions)
	if len(permissions) == 0 {
		return domain.AdminRole{}, errors.New("at least one permission is required")
	}

	updates := map[string]any{
		"name":        name,
		"description": strings.TrimSpace(input.Description),
		"permissions": storage.JSONStringSlice(permissions),
		"sort":        input.Sort,
	}
	if err := s.db.WithContext(ctx).Model(&model).Updates(updates).Error; err != nil {
		return domain.AdminRole{}, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", roleID).First(&model).Error; err != nil {
		return domain.AdminRole{}, err
	}
	return toAdminRole(model), nil
}

func (s *Service) RoleHasPermission(ctx context.Context, roleKey, permission string) (bool, error) {
	roleKey = strings.TrimSpace(roleKey)
	permission = strings.TrimSpace(permission)
	if roleKey == "" || permission == "" {
		return false, nil
	}

	var role storage.AdminRole
	if err := s.db.WithContext(ctx).Where("role_key = ?", roleKey).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	for _, item := range role.Permissions {
		if item == "*" || item == permission {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) EnsureInitialImport(ctx context.Context, subjectKey, path string) (domain.ImportResult, bool, error) {
	subject, err := s.ensureSubject(ctx, subjectKey)
	if err != nil {
		return domain.ImportResult{}, false, err
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&storage.Word{}).Where("subject_id = ?", subject.ID).Count(&count).Error; err != nil {
		return domain.ImportResult{}, false, err
	}
	if count > 0 {
		return domain.ImportResult{
			ImportedCount: 0,
			SubjectKey:    subjectKey,
			Path:          path,
			Replace:       false,
		}, false, nil
	}

	result, err := s.ImportWordsFromFile(ctx, domain.ImportWordsInput{
		Path:       path,
		SubjectKey: subjectKey,
		Replace:    boolPtr(true),
	})
	return result, true, err
}

func (s *Service) ListSubjects(ctx context.Context) ([]domain.Subject, error) {
	var models []storage.Subject
	if err := s.db.WithContext(ctx).Order("sort asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]domain.Subject, 0, len(models))
	for _, model := range models {
		items = append(items, toSubject(model))
	}
	return items, nil
}

func (s *Service) ListCategories(ctx context.Context, subjectKey, kind string) ([]domain.Category, error) {
	query := s.db.WithContext(ctx).Model(&storage.Category{})
	subjectKey = strings.TrimSpace(subjectKey)
	if subjectKey != "" {
		var subject storage.Subject
		if err := s.db.WithContext(ctx).Where("subject_key = ?", subjectKey).First(&subject).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return []domain.Category{}, nil
			}
			return nil, err
		}
		query = query.Where("subject_id = ?", subject.ID)
	}

	kind = strings.TrimSpace(kind)
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}

	var models []storage.Category
	if err := query.Order("sort asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}

	subjectLookup, err := s.subjectKeyLookup(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]domain.Category, 0, len(models))
	for _, model := range models {
		items = append(items, toCategory(model, subjectLookup[model.SubjectID]))
	}
	return items, nil
}

func (s *Service) ListCategoriesPaged(ctx context.Context, filter domain.CategoryFilter) (domain.PagedCategories, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).Model(&storage.Category{})

	subjectKey := strings.TrimSpace(filter.SubjectKey)
	if subjectKey != "" {
		var subject storage.Subject
		if err := s.db.WithContext(ctx).Where("subject_key = ?", subjectKey).First(&subject).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.PagedCategories{Items: []domain.Category{}, Total: 0, Page: page, PageSize: pageSize}, nil
			}
			return domain.PagedCategories{}, err
		}
		query = query.Where("subject_id = ?", subject.ID)
	}

	if kind := strings.TrimSpace(filter.Kind); kind != "" {
		query = query.Where("kind = ?", kind)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR category_key LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedCategories{}, err
	}

	var models []storage.Category
	if err := query.Order("sort asc, id asc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedCategories{}, err
	}

	subjectLookup, err := s.subjectKeyLookup(ctx)
	if err != nil {
		return domain.PagedCategories{}, err
	}

	items := make([]domain.Category, 0, len(models))
	for _, model := range models {
		items = append(items, toCategory(model, subjectLookup[model.SubjectID]))
	}

	return domain.PagedCategories{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ListGrades(ctx context.Context) ([]domain.Grade, error) {
	var models []storage.Grade
	if err := s.db.WithContext(ctx).Order("sort asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]domain.Grade, 0, len(models))
	for _, model := range models {
		items = append(items, toGrade(model))
	}
	return items, nil
}

func (s *Service) ListGradesPaged(ctx context.Context, filter domain.GradeFilter) (domain.PagedGrades, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).Model(&storage.Grade{})

	if stage := strings.TrimSpace(filter.Stage); stage != "" {
		query = query.Where("stage = ?", stage)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR grade_key LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedGrades{}, err
	}

	var models []storage.Grade
	if err := query.Order("sort asc, id asc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedGrades{}, err
	}

	items := make([]domain.Grade, 0, len(models))
	for _, model := range models {
		items = append(items, toGrade(model))
	}

	return domain.PagedGrades{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	var models []storage.Plan
	if err := s.db.WithContext(ctx).Order("recommended desc, price_cents asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]domain.Plan, 0, len(models))
	for _, model := range models {
		items = append(items, toPlan(model))
	}
	return items, nil
}

func (s *Service) ListWords(ctx context.Context, filter domain.WordFilter) (domain.PagedWords, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := s.db.WithContext(ctx).
		Model(&storage.Word{}).
		Preload("Subject").
		Preload("Category").
		Preload("Grade")

	if filter.SubjectID > 0 {
		query = query.Where("subject_id = ?", filter.SubjectID)
	} else if strings.TrimSpace(filter.SubjectKey) != "" {
		query = query.Joins("JOIN subjects ON subjects.id = words.subject_id").Where("subjects.subject_key = ?", strings.TrimSpace(filter.SubjectKey))
	}

	if filter.CategoryID > 0 {
		query = query.Where("category_id = ?", filter.CategoryID)
	}

	if strings.TrimSpace(filter.Classification) != "" {
		query = query.Joins("LEFT JOIN categories ON categories.id = words.category_id").Where("categories.name = ?", strings.TrimSpace(filter.Classification))
	}

	if filter.GradeID > 0 {
		query = query.Where("grade_id = ?", filter.GradeID)
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"term LIKE ? OR translation LIKE ? OR phonetics LIKE ? OR explanation LIKE ? OR source_label LIKE ?",
			like, like, like, like, like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedWords{}, err
	}

	var models []storage.Word
	if err := query.
		Order("words.id asc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&models).Error; err != nil {
		return domain.PagedWords{}, err
	}

	items := make([]domain.Word, 0, len(models))
	for _, model := range models {
		items = append(items, toWord(model))
	}

	return domain.PagedWords{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ListClassificationStats(ctx context.Context, subjectKey string) ([]domain.ClassificationStat, error) {
	query := s.db.WithContext(ctx).Table("words").
		Select("COALESCE(categories.name, ?) AS name, COUNT(words.id) AS count", "Unclassified").
		Joins("LEFT JOIN categories ON categories.id = words.category_id")

	if strings.TrimSpace(subjectKey) != "" {
		query = query.Joins("JOIN subjects ON subjects.id = words.subject_id").Where("subjects.subject_key = ?", strings.TrimSpace(subjectKey))
	}

	type row struct {
		Name  string
		Count int
	}
	var rows []row
	if err := query.Group("COALESCE(categories.name, 'Unclassified')").Order("count desc, name asc").Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]domain.ClassificationStat, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.ClassificationStat{Name: row.Name, Count: row.Count})
	}
	return items, nil
}

func (s *Service) Stats(ctx context.Context) (domain.CatalogStats, error) {
	var stats domain.CatalogStats
	if err := s.db.WithContext(ctx).Model(&storage.Subject{}).Count(&stats.SubjectCount).Error; err != nil {
		return stats, err
	}
	if err := s.db.WithContext(ctx).Model(&storage.Word{}).Count(&stats.WordCount).Error; err != nil {
		return stats, err
	}
	if err := s.db.WithContext(ctx).Model(&storage.Category{}).Where("kind = ?", topicCategoryKind).Count(&stats.ClassificationCount).Error; err != nil {
		return stats, err
	}
	if err := s.db.WithContext(ctx).Model(&storage.Grade{}).Count(&stats.GradeCount).Error; err != nil {
		return stats, err
	}
	if err := s.db.WithContext(ctx).Model(&storage.AdminUser{}).Count(&stats.AdminCount).Error; err != nil {
		return stats, err
	}

	var latest storage.ImportJob
	err := s.db.WithContext(ctx).
		Where("status = ?", "success").
		Order("finished_at desc, id desc").
		First(&latest).Error
	if err == nil {
		stats.DataSource = latest.SourceName
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return stats, err
	}

	stats.SampleData = stats.WordCount == 0
	stats.SuperAdminInitialized = stats.AdminCount > 0
	return stats, nil
}

func (s *Service) CreateSubject(ctx context.Context, input domain.CreateSubjectInput) (domain.Subject, error) {
	model := storage.Subject{
		Key:         normalizeKey(input.Key),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Sort:        input.Sort,
		Featured:    input.Featured,
	}
	if model.Key == "" || model.Name == "" {
		return domain.Subject{}, errors.New("subject key and name are required")
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Subject{}, err
	}
	return toSubject(model), nil
}

func (s *Service) CreateCategory(ctx context.Context, input domain.CreateCategoryInput) (domain.Category, error) {
	subject, err := s.resolveSubject(ctx, input.SubjectID, input.SubjectKey)
	if err != nil {
		return domain.Category{}, err
	}

	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = topicCategoryKind
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Category{}, errors.New("category name is required")
	}

	key := strings.TrimSpace(input.Key)
	if key == "" {
		key = stableKey(name)
	}

	model := storage.Category{
		SubjectID:   subject.ID,
		Kind:        kind,
		Key:         key,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Sort:        input.Sort,
		Enabled:     boolOrDefault(input.Enabled, true),
	}

	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Category{}, err
	}
	return toCategory(model, subject.Key), nil
}

func (s *Service) CreateGrade(ctx context.Context, input domain.CreateGradeInput) (domain.Grade, error) {
	model := storage.Grade{
		Key:         normalizeKey(input.Key),
		Name:        strings.TrimSpace(input.Name),
		Stage:       strings.TrimSpace(input.Stage),
		Description: strings.TrimSpace(input.Description),
		Sort:        input.Sort,
		Enabled:     boolOrDefault(input.Enabled, true),
	}
	if model.Key == "" || model.Name == "" {
		return domain.Grade{}, errors.New("grade key and name are required")
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Grade{}, err
	}
	return toGrade(model), nil
}

func (s *Service) CreateWord(ctx context.Context, input domain.CreateWordInput) (domain.Word, error) {
	subject, err := s.resolveSubject(ctx, input.SubjectID, input.SubjectKey)
	if err != nil {
		return domain.Word{}, err
	}

	term := strings.TrimSpace(input.Term)
	if term == "" {
		return domain.Word{}, errors.New("term is required")
	}

	var categoryID *uint
	categoryName := strings.TrimSpace(input.CategoryName)
	if categoryName == "" {
		categoryName = strings.TrimSpace(input.Classification)
	}
	if input.CategoryID != nil && *input.CategoryID > 0 {
		categoryID = input.CategoryID
	} else if categoryName != "" {
		category, err := s.findOrCreateTopicCategory(ctx, subject.ID, categoryName)
		if err != nil {
			return domain.Word{}, err
		}
		categoryID = &category.ID
	}

	var gradeID *uint
	if input.GradeID != nil && *input.GradeID > 0 {
		gradeID = input.GradeID
	}

	model := storage.Word{
		LegacyID:    input.LegacyID,
		SubjectID:   subject.ID,
		CategoryID:  categoryID,
		GradeID:     gradeID,
		Term:        term,
		Translation: strings.TrimSpace(input.Translation),
		SourceLabel: strings.TrimSpace(input.Source),
		Phonetics:   strings.TrimSpace(input.Phonetics),
		Explanation: strings.TrimSpace(input.Explanation),
		IsVIP:       input.IsVIP,
		Status:      "published",
	}

	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Word{}, err
	}

	if err := s.db.WithContext(ctx).
		Preload("Subject").
		Preload("Category").
		Preload("Grade").
		Where("id = ?", model.ID).
		First(&model).Error; err != nil {
		return domain.Word{}, err
	}

	return toWord(model), nil
}

func (s *Service) CreatePlan(ctx context.Context, input domain.CreatePlanInput) (domain.Plan, error) {
	model := storage.Plan{
		Key:             normalizeKey(input.Key),
		Name:            strings.TrimSpace(input.Name),
		BillingMode:     normalizeBillingMode(input.BillingMode),
		PriceCents:      input.PriceCents,
		Description:     strings.TrimSpace(input.Description),
		Recommended:     input.Recommended,
		PaymentChannels: storage.JSONStringSlice(cleanStringSlice(input.PaymentChannels)),
		Features:        storage.JSONStringSlice(cleanStringSlice(input.Features)),
	}
	if model.Key == "" || model.Name == "" {
		return domain.Plan{}, errors.New("plan key and name are required")
	}
	if model.PriceCents < 0 {
		return domain.Plan{}, errors.New("plan price must be greater than or equal to 0")
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Plan{}, err
	}
	return toPlan(model), nil
}

func (s *Service) UpdatePlan(ctx context.Context, id uint, input domain.UpdatePlanInput) (domain.Plan, error) {
	if id == 0 {
		return domain.Plan{}, errors.New("plan id is required")
	}

	var model storage.Plan
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Plan{}, errors.New("plan does not exist")
		}
		return domain.Plan{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Plan{}, errors.New("plan name is required")
	}
	if input.PriceCents < 0 {
		return domain.Plan{}, errors.New("plan price must be greater than or equal to 0")
	}

	model.Name = name
	model.BillingMode = normalizeBillingMode(input.BillingMode)
	model.PriceCents = input.PriceCents
	model.Description = strings.TrimSpace(input.Description)
	model.Recommended = input.Recommended
	model.PaymentChannels = storage.JSONStringSlice(cleanStringSlice(input.PaymentChannels))
	model.Features = storage.JSONStringSlice(cleanStringSlice(input.Features))

	if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
		return domain.Plan{}, err
	}
	return toPlan(model), nil
}

func (s *Service) DeletePlan(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("plan id is required")
	}

	var model storage.Plan
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("plan does not exist")
		}
		return err
	}

	var orderCount int64
	if err := s.db.WithContext(ctx).Model(&storage.PaymentOrder{}).
		Where("plan_id = ? OR plan_key = ?", model.ID, model.Key).
		Count(&orderCount).Error; err != nil {
		return err
	}
	if orderCount > 0 {
		return errors.New("plan already has payment orders and cannot be deleted")
	}

	var subscriptionCount int64
	if err := s.db.WithContext(ctx).Model(&storage.MemberSubscription{}).
		Where("plan_id = ? OR plan_key = ?", model.ID, model.Key).
		Count(&subscriptionCount).Error; err != nil {
		return err
	}
	if subscriptionCount > 0 {
		return errors.New("plan already has member subscriptions and cannot be deleted")
	}

	return s.db.WithContext(ctx).Delete(&model).Error
}

func (s *Service) ImportWordsFromFile(ctx context.Context, input domain.ImportWordsInput) (domain.ImportResult, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return domain.ImportResult{}, errors.New("path is required")
	}

	subjectKey := strings.TrimSpace(input.SubjectKey)
	if subjectKey == "" {
		subjectKey = "english"
	}
	replace := boolOrDefault(input.Replace, true)

	subject, err := s.ensureSubject(ctx, subjectKey)
	if err != nil {
		return domain.ImportResult{}, err
	}

	rows, err := catalog.LoadWordsFromFile(path, subjectKey)
	if err != nil {
		return domain.ImportResult{}, err
	}

	job := storage.ImportJob{
		SubjectID:     subject.ID,
		SourcePath:    path,
		SourceName:    filepath.Base(path),
		ReplaceMode:   replace,
		Status:        "running",
		ImportedCount: 0,
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return domain.ImportResult{}, err
	}

	var createdCategories int
	importErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		categoryByName, count, err := ensureImportCategories(tx, subject.ID, rows)
		if err != nil {
			return err
		}
		createdCategories = count

		if replace {
			if err := tx.Where("subject_id = ?", subject.ID).Delete(&storage.Word{}).Error; err != nil {
				return err
			}
		}

		batch := make([]storage.Word, 0, 1000)
		flush := func() error {
			if len(batch) == 0 {
				return nil
			}
			if err := tx.CreateInBatches(batch, 1000).Error; err != nil {
				return err
			}
			batch = batch[:0]
			return nil
		}

		for _, item := range rows {
			record := storage.Word{
				LegacyID:    item.LegacyID,
				SubjectID:   subject.ID,
				Term:        strings.TrimSpace(item.Term),
				Translation: strings.TrimSpace(item.Translation),
				SourceLabel: strings.TrimSpace(item.Source),
				Phonetics:   strings.TrimSpace(item.Phonetics),
				Explanation: strings.TrimSpace(item.Explanation),
				IsVIP:       false,
				Status:      "published",
			}
			if category, ok := categoryByName[item.Classification]; ok {
				record.CategoryID = uintPtr(category.ID)
			}
			batch = append(batch, record)
			if len(batch) >= 1000 {
				if err := flush(); err != nil {
					return err
				}
			}
		}

		return flush()
	})

	finishedAt := time.Now()
	if importErr != nil {
		_ = s.db.WithContext(ctx).Model(&job).Updates(map[string]any{
			"status":        "failed",
			"error_message": importErr.Error(),
			"finished_at":   &finishedAt,
		}).Error
		return domain.ImportResult{}, importErr
	}

	if err := s.db.WithContext(ctx).Model(&job).Updates(map[string]any{
		"status":         "success",
		"imported_count": len(rows),
		"finished_at":    &finishedAt,
	}).Error; err != nil {
		return domain.ImportResult{}, err
	}

	return domain.ImportResult{
		ImportedCount:     len(rows),
		CreatedCategories: createdCategories,
		SubjectKey:        subjectKey,
		Path:              path,
		Replace:           replace,
	}, nil
}

func ensureImportCategories(tx *gorm.DB, subjectID uint, rows []domain.Word) (map[string]storage.Category, int, error) {
	var existing []storage.Category
	if err := tx.Where("subject_id = ? AND kind = ?", subjectID, topicCategoryKind).Find(&existing).Error; err != nil {
		return nil, 0, err
	}

	categoryByName := make(map[string]storage.Category, len(existing))
	keyUsage := make(map[string]int, len(existing))
	for _, item := range existing {
		categoryByName[item.Name] = item
		keyUsage[item.Key]++
	}

	uniqueNames := make([]string, 0, 64)
	seen := make(map[string]struct{})
	for _, row := range rows {
		name := strings.TrimSpace(row.Classification)
		if name == "" {
			name = "Unclassified"
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		uniqueNames = append(uniqueNames, name)
	}
	sort.Strings(uniqueNames)

	missing := make([]storage.Category, 0)
	for _, name := range uniqueNames {
		if _, ok := categoryByName[name]; ok {
			continue
		}
		key := stableKey(name)
		if key == "" {
			key = "topic"
		}
		if keyUsage[key] > 0 {
			key = fmt.Sprintf("%s-%d", key, keyUsage[key]+1)
		}
		keyUsage[key]++
		missing = append(missing, storage.Category{
			SubjectID: subjectID,
			Kind:      topicCategoryKind,
			Key:       key,
			Name:      name,
			Enabled:   true,
		})
	}

	if len(missing) > 0 {
		if err := tx.Create(&missing).Error; err != nil {
			return nil, 0, err
		}
		for _, item := range missing {
			categoryByName[item.Name] = item
		}
	}

	return categoryByName, len(missing), nil
}

func (s *Service) ensureSubject(ctx context.Context, key string) (storage.Subject, error) {
	key = normalizeKey(key)
	if key == "" {
		return storage.Subject{}, errors.New("subject key is required")
	}

	var subject storage.Subject
	err := s.db.WithContext(ctx).Where("subject_key = ?", key).First(&subject).Error
	if err == nil {
		return subject, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return storage.Subject{}, err
	}

	if key != "english" {
		return storage.Subject{}, errors.New("subject does not exist")
	}

	subject = storage.Subject{
		Key:         "english",
		Name:        "英语高频词汇",
		Description: "围绕真实场景整理的高频英语单词与学习内容。",
		Sort:        1,
		Featured:    true,
	}
	if err := s.db.WithContext(ctx).Create(&subject).Error; err != nil {
		return storage.Subject{}, err
	}
	return subject, nil
}

func (s *Service) resolveSubject(ctx context.Context, subjectID uint, subjectKey string) (storage.Subject, error) {
	if subjectID > 0 {
		var subject storage.Subject
		if err := s.db.WithContext(ctx).Where("id = ?", subjectID).First(&subject).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.Subject{}, errors.New("subject does not exist")
			}
			return storage.Subject{}, err
		}
		return subject, nil
	}
	return s.ensureSubject(ctx, subjectKey)
}

func (s *Service) findOrCreateTopicCategory(ctx context.Context, subjectID uint, name string) (storage.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return storage.Category{}, errors.New("category name is required")
	}

	var category storage.Category
	err := s.db.WithContext(ctx).Where("subject_id = ? AND kind = ? AND name = ?", subjectID, topicCategoryKind, name).First(&category).Error
	if err == nil {
		return category, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return storage.Category{}, err
	}

	category = storage.Category{
		SubjectID: subjectID,
		Kind:      topicCategoryKind,
		Key:       stableKey(name),
		Name:      name,
		Enabled:   true,
	}
	if err := s.db.WithContext(ctx).Create(&category).Error; err != nil {
		return storage.Category{}, err
	}
	return category, nil
}

func (s *Service) subjectKeyLookup(ctx context.Context) (map[uint]string, error) {
	var subjects []storage.Subject
	if err := s.db.WithContext(ctx).Find(&subjects).Error; err != nil {
		return nil, err
	}
	lookup := make(map[uint]string, len(subjects))
	for _, subject := range subjects {
		lookup[subject.ID] = subject.Key
	}
	return lookup, nil
}

func (s *Service) upsertSubject(ctx context.Context, model storage.Subject) error {
	var existing storage.Subject
	err := s.db.WithContext(ctx).Where("subject_key = ?", model.Key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.WithContext(ctx).Create(&model).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"name":        model.Name,
		"description": model.Description,
		"sort":        model.Sort,
		"featured":    model.Featured,
	}).Error
}

func (s *Service) upsertPlan(ctx context.Context, model storage.Plan) error {
	var existing storage.Plan
	err := s.db.WithContext(ctx).Where("plan_key = ?", model.Key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.WithContext(ctx).Create(&model).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"name":             model.Name,
		"billing_mode":     model.BillingMode,
		"price_cents":      model.PriceCents,
		"description":      model.Description,
		"recommended":      model.Recommended,
		"payment_channels": model.PaymentChannels,
		"features":         model.Features,
	}).Error
}

func (s *Service) upsertAdminRole(ctx context.Context, model storage.AdminRole) error {
	var existing storage.AdminRole
	err := s.db.WithContext(ctx).Where("role_key = ?", model.Key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.WithContext(ctx).Create(&model).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"name":        model.Name,
		"description": model.Description,
		"permissions": model.Permissions,
		"system":      model.System,
		"sort":        model.Sort,
	}).Error
}

func (s *Service) ensureAdminRoleExists(ctx context.Context, roleKey string) error {
	roleKey = normalizeRoleKey(roleKey)
	if roleKey == "" {
		return errors.New("role is required")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&storage.AdminRole{}).Where("role_key = ?", roleKey).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("role does not exist")
	}
	return nil
}

func (s *Service) countActiveSuperAdmins(ctx context.Context, excludeID uint) (int64, error) {
	query := s.db.WithContext(ctx).Model(&storage.AdminUser{}).Where("is_super = ? AND status = ?", true, "active")
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func toSubject(model storage.Subject) domain.Subject {
	return domain.Subject{
		ID:          model.ID,
		Key:         model.Key,
		Name:        model.Name,
		Description: model.Description,
		Sort:        model.Sort,
		Featured:    model.Featured,
	}
}

func toCategory(model storage.Category, subjectKey string) domain.Category {
	return domain.Category{
		ID:          model.ID,
		SubjectID:   model.SubjectID,
		SubjectKey:  subjectKey,
		Kind:        model.Kind,
		Key:         model.Key,
		Name:        model.Name,
		Description: model.Description,
		Sort:        model.Sort,
		Enabled:     model.Enabled,
	}
}

func toGrade(model storage.Grade) domain.Grade {
	return domain.Grade{
		ID:          model.ID,
		Key:         model.Key,
		Name:        model.Name,
		Stage:       model.Stage,
		Description: model.Description,
		Sort:        model.Sort,
		Enabled:     model.Enabled,
	}
}

func toWord(model storage.Word) domain.Word {
	item := domain.Word{
		ID:          model.ID,
		LegacyID:    model.LegacyID,
		SubjectID:   model.SubjectID,
		Term:        model.Term,
		Translation: model.Translation,
		Source:      model.SourceLabel,
		Phonetics:   model.Phonetics,
		Explanation: model.Explanation,
		IsVIP:       model.IsVIP,
	}
	if model.Subject.ID > 0 {
		item.SubjectKey = model.Subject.Key
	}
	if model.Category != nil {
		item.CategoryID = uintPtr(model.Category.ID)
		item.CategoryName = model.Category.Name
		item.Classification = model.Category.Name
	}
	if item.Classification == "" {
		item.Classification = "Unclassified"
	}
	if model.Grade != nil {
		item.GradeID = uintPtr(model.Grade.ID)
		item.GradeName = model.Grade.Name
	}
	return item
}

func toPlan(model storage.Plan) domain.Plan {
	return domain.Plan{
		ID:              model.ID,
		Key:             model.Key,
		Name:            model.Name,
		BillingMode:     model.BillingMode,
		PriceCents:      model.PriceCents,
		Description:     model.Description,
		Recommended:     model.Recommended,
		PaymentChannels: []string(model.PaymentChannels),
		Features:        []string(model.Features),
	}
}

func toAdminUser(model storage.AdminUser) domain.AdminUser {
	return domain.AdminUser{
		ID:          model.ID,
		Username:    model.Username,
		DisplayName: model.DisplayName,
		Role:        model.Role,
		IsSuper:     model.IsSuper,
		Status:      model.Status,
		LastLoginAt: model.LastLoginAt,
	}
}

func toAdminRole(model storage.AdminRole) domain.AdminRole {
	return domain.AdminRole{
		ID:          model.ID,
		Key:         model.Key,
		Name:        model.Name,
		Description: model.Description,
		Permissions: []string(model.Permissions),
		System:      model.System,
		Sort:        model.Sort,
	}
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func cleanStringSlice(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func normalizeBillingMode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "monthly":
		return "monthly"
	default:
		return "lifetime"
	}
}

func cleanPermissionSet(items []string) []string {
	cleaned := cleanStringSlice(items)
	seen := make(map[string]struct{}, len(cleaned))
	result := make([]string, 0, len(cleaned))
	for _, item := range cleaned {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func normalizeAdminStatus(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "active", nil
	}
	switch value {
	case "active", "disabled":
		return value, nil
	default:
		return "", errors.New("status must be active or disabled")
	}
}

func normalizePage(page, pageSize, defaultPageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", "_", "-", ".", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func normalizeRoleKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ".", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func stableKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	trimmed := value
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	trimmed = strings.ReplaceAll(trimmed, "/", "-")
	trimmed = strings.ReplaceAll(trimmed, "\\", "-")
	trimmed = strings.Trim(trimmed, "-")
	if len([]rune(trimmed)) <= 100 {
		return trimmed
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(trimmed))
	runes := []rune(trimmed)
	return string(runes[:80]) + "-" + fmt.Sprintf("%x", hash.Sum32())
}

func uintPtr(value uint) *uint {
	v := value
	return &v
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
