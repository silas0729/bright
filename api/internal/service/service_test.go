package service

import (
	"context"
	"fmt"
	"testing"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	db, err := storage.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()), false)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := storage.AutoMigrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return New(db)
}

func TestBootstrapSuperAdminIsIdempotent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, created, err := svc.BootstrapSuperAdmin(ctx, domain.BootstrapAdminInput{
		Username:    "superadmin",
		Password:    "ChangeMe@123456",
		DisplayName: "Root User",
	})
	if err != nil {
		t.Fatalf("first bootstrap failed: %v", err)
	}
	if !created {
		t.Fatal("expected first bootstrap to create user")
	}

	user, created, err := svc.BootstrapSuperAdmin(ctx, domain.BootstrapAdminInput{
		Username:    "superadmin",
		Password:    "Reset@123456",
		DisplayName: "Updated Root",
	})
	if err != nil {
		t.Fatalf("second bootstrap failed: %v", err)
	}
	if created {
		t.Fatal("expected second bootstrap to update existing user")
	}
	if user.DisplayName != "Updated Root" {
		t.Fatalf("expected display name update, got %q", user.DisplayName)
	}
}

func TestCreateWordCreatesTopicCategory(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	word, err := svc.CreateWord(ctx, domain.CreateWordInput{
		SubjectKey:     "english",
		Classification: "driving",
		Term:           "pedal",
		Translation:    "踏板",
	})
	if err != nil {
		t.Fatalf("create word: %v", err)
	}
	if word.CategoryName != "driving" {
		t.Fatalf("expected category driving, got %q", word.CategoryName)
	}
}

func TestEnsureClassificationSummariesBackfillsAndPages(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	subject, err := svc.ensureSubject(ctx, "english")
	if err != nil {
		t.Fatalf("ensure subject: %v", err)
	}

	finance, err := svc.findOrCreateTopicCategory(ctx, subject.ID, "finance")
	if err != nil {
		t.Fatalf("create finance category: %v", err)
	}
	travel, err := svc.findOrCreateTopicCategory(ctx, subject.ID, "travel")
	if err != nil {
		t.Fatalf("create travel category: %v", err)
	}

	seedWords := []storage.Word{
		{SubjectID: subject.ID, CategoryID: uintPtr(finance.ID), Term: "budget", Translation: "预算", Status: "published"},
		{SubjectID: subject.ID, CategoryID: uintPtr(finance.ID), Term: "invoice", Translation: "发票", Status: "published"},
		{SubjectID: subject.ID, CategoryID: uintPtr(finance.ID), Term: "profit", Translation: "利润", Status: "published"},
		{SubjectID: subject.ID, CategoryID: uintPtr(travel.ID), Term: "boarding", Translation: "登机", Status: "published"},
		{SubjectID: subject.ID, CategoryID: uintPtr(travel.ID), Term: "luggage", Translation: "行李", Status: "published"},
		{SubjectID: subject.ID, Term: "context", Translation: "语境", Status: "published"},
	}
	if err := svc.db.WithContext(ctx).Create(&seedWords).Error; err != nil {
		t.Fatalf("seed words: %v", err)
	}

	if err := svc.EnsureClassificationSummaries(ctx); err != nil {
		t.Fatalf("ensure classification summaries: %v", err)
	}

	var summaryCount int64
	if err := svc.db.WithContext(ctx).Model(&storage.ClassificationSummary{}).Where("subject_id = ?", subject.ID).Count(&summaryCount).Error; err != nil {
		t.Fatalf("count summaries: %v", err)
	}
	if summaryCount != 3 {
		t.Fatalf("expected 3 classification summaries, got %d", summaryCount)
	}

	pageOne, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   2,
	})
	if err != nil {
		t.Fatalf("list classification page 1: %v", err)
	}
	if pageOne.Total != 3 {
		t.Fatalf("expected total 3, got %d", pageOne.Total)
	}
	if len(pageOne.Items) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(pageOne.Items))
	}
	if pageOne.Items[0].Name != "finance" || pageOne.Items[0].Count != 3 {
		t.Fatalf("unexpected first item: %+v", pageOne.Items[0])
	}
	if pageOne.Items[1].Name != "travel" || pageOne.Items[1].Count != 2 {
		t.Fatalf("unexpected second item: %+v", pageOne.Items[1])
	}

	pageTwo, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       2,
		PageSize:   2,
	})
	if err != nil {
		t.Fatalf("list classification page 2: %v", err)
	}
	if len(pageTwo.Items) != 1 {
		t.Fatalf("expected 1 item on page 2, got %d", len(pageTwo.Items))
	}
	if pageTwo.Items[0].Name != "Unclassified" || pageTwo.Items[0].Count != 1 {
		t.Fatalf("unexpected page 2 item: %+v", pageTwo.Items[0])
	}
}

func TestUpdateCategoryRefreshesClassificationSummaries(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	word, err := svc.CreateWord(ctx, domain.CreateWordInput{
		SubjectKey:     "english",
		Classification: "finance",
		Term:           "invoice",
		Translation:    "发票",
	})
	if err != nil {
		t.Fatalf("create word: %v", err)
	}
	if word.CategoryID == nil || *word.CategoryID == 0 {
		t.Fatal("expected word category id")
	}

	enabled := true
	updated, err := svc.UpdateCategory(ctx, *word.CategoryID, domain.UpdateCategoryInput{
		SubjectKey:  "english",
		Kind:        "topic",
		Key:         "business-finance",
		Name:        "business finance",
		Description: "Business finance terms",
		Sort:        2,
		Enabled:     &enabled,
	})
	if err != nil {
		t.Fatalf("update category: %v", err)
	}
	if updated.Name != "business finance" {
		t.Fatalf("expected updated category name, got %q", updated.Name)
	}

	stats, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list classification stats: %v", err)
	}
	if len(stats.Items) != 1 {
		t.Fatalf("expected 1 classification item, got %d", len(stats.Items))
	}
	if stats.Items[0].Name != "business finance" || stats.Items[0].Count != 1 {
		t.Fatalf("unexpected classification summary: %+v", stats.Items[0])
	}
}

func TestUpdateWordMovesSubjectAndRebuildsSummaries(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	_, err := svc.CreateSubject(ctx, domain.CreateSubjectInput{
		Key:         "science",
		Name:        "Science",
		Description: "Science subject",
		Sort:        2,
		Featured:    false,
	})
	if err != nil {
		t.Fatalf("create subject: %v", err)
	}

	word, err := svc.CreateWord(ctx, domain.CreateWordInput{
		SubjectKey:     "english",
		Classification: "travel",
		Term:           "boarding",
		Translation:    "登机",
	})
	if err != nil {
		t.Fatalf("create word: %v", err)
	}

	updated, err := svc.UpdateWord(ctx, word.ID, domain.UpdateWordInput{
		SubjectKey:     "science",
		Classification: "physics",
		Term:           "atom",
		Translation:    "原子",
		Source:         "manual",
		Phonetics:      "ˈætəm",
		Explanation:    "A basic unit of matter.",
		IsVIP:          true,
	})
	if err != nil {
		t.Fatalf("update word: %v", err)
	}
	if updated.SubjectKey != "science" {
		t.Fatalf("expected updated subject science, got %q", updated.SubjectKey)
	}
	if updated.Classification != "physics" {
		t.Fatalf("expected updated classification physics, got %q", updated.Classification)
	}

	englishStats, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list english classification stats: %v", err)
	}
	if englishStats.Total != 0 {
		t.Fatalf("expected english classification total 0, got %d", englishStats.Total)
	}

	scienceStats, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "science",
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list science classification stats: %v", err)
	}
	if scienceStats.Total != 1 {
		t.Fatalf("expected science classification total 1, got %d", scienceStats.Total)
	}
	if len(scienceStats.Items) != 1 || scienceStats.Items[0].Name != "physics" || scienceStats.Items[0].Count != 1 {
		t.Fatalf("unexpected science classification stats: %+v", scienceStats.Items)
	}
}

func TestAuthenticateAdmin(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, _, err := svc.BootstrapSuperAdmin(ctx, domain.BootstrapAdminInput{
		Username:    "superadmin",
		Password:    "ChangeMe@123456",
		DisplayName: "Root",
	})
	if err != nil {
		t.Fatalf("bootstrap super admin: %v", err)
	}

	admin, err := svc.AuthenticateAdmin(ctx, "superadmin", "ChangeMe@123456")
	if err != nil {
		t.Fatalf("authenticate admin: %v", err)
	}
	if admin.Username != "superadmin" {
		t.Fatalf("expected username superadmin, got %q", admin.Username)
	}
}

func TestCreateAndUpdateAdminUser(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	root, _, err := svc.BootstrapSuperAdmin(ctx, domain.BootstrapAdminInput{
		Username:    "superadmin",
		Password:    "ChangeMe@123456",
		DisplayName: "Root",
	})
	if err != nil {
		t.Fatalf("bootstrap super admin: %v", err)
	}

	admin, err := svc.CreateAdminUser(ctx, domain.CreateAdminUserInput{
		Username:    "content-manager",
		Password:    "Manager@123",
		DisplayName: "Content Manager",
		Role:        "content_admin",
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if admin.Role != "content_admin" {
		t.Fatalf("expected role content_admin, got %q", admin.Role)
	}

	updated, err := svc.UpdateAdminUser(ctx, admin.ID, root.ID, domain.UpdateAdminUserInput{
		DisplayName: "Content Manager Updated",
		Status:      "disabled",
	})
	if err != nil {
		t.Fatalf("update admin user: %v", err)
	}
	if updated.Status != "disabled" {
		t.Fatalf("expected disabled status, got %q", updated.Status)
	}
}

func TestCannotDisableLastSuperAdmin(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	root, _, err := svc.BootstrapSuperAdmin(ctx, domain.BootstrapAdminInput{
		Username:    "superadmin",
		Password:    "ChangeMe@123456",
		DisplayName: "Root",
	})
	if err != nil {
		t.Fatalf("bootstrap super admin: %v", err)
	}

	_, err = svc.UpdateAdminUser(ctx, root.ID, root.ID, domain.UpdateAdminUserInput{
		Status: "disabled",
	})
	if err == nil {
		t.Fatal("expected disabling last super admin to fail")
	}
}

func TestCreateAndUpdateCustomAdminRole(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	role, err := svc.CreateAdminRole(ctx, domain.CreateAdminRoleInput{
		Key:         "ops_manager",
		Name:        "Ops Manager",
		Description: "Operations role",
		Permissions: []string{"admin.read", "catalog.read"},
		Sort:        10,
	})
	if err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if role.Key != "ops_manager" {
		t.Fatalf("expected normalized role key ops_manager, got %q", role.Key)
	}

	updated, err := svc.UpdateAdminRole(ctx, role.ID, domain.UpdateAdminRoleInput{
		Name:        "Ops Manager Updated",
		Description: "Updated operations role",
		Permissions: []string{"admin.read", "catalog.read", "grade.read"},
		Sort:        20,
	})
	if err != nil {
		t.Fatalf("update admin role: %v", err)
	}
	if updated.Name != "Ops Manager Updated" {
		t.Fatalf("expected updated role name, got %q", updated.Name)
	}
}

func TestResetSuperAdminPassword(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	root, _, err := svc.BootstrapSuperAdmin(ctx, domain.BootstrapAdminInput{
		Username:    "superadmin",
		Password:    "ChangeMe@123456",
		DisplayName: "Root",
	})
	if err != nil {
		t.Fatalf("bootstrap super admin: %v", err)
	}

	updated, err := svc.ResetSuperAdminPassword(ctx, root.Username, "Reset@123456", "站点管理员")
	if err != nil {
		t.Fatalf("reset super admin password: %v", err)
	}
	if updated.DisplayName != "站点管理员" {
		t.Fatalf("expected updated display name, got %q", updated.DisplayName)
	}

	admin, err := svc.AuthenticateAdmin(ctx, root.Username, "Reset@123456")
	if err != nil {
		t.Fatalf("authenticate with reset password: %v", err)
	}
	if admin.Username != root.Username {
		t.Fatalf("expected username %q, got %q", root.Username, admin.Username)
	}
}

func TestRegisterAndAuthenticateLearner(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	user, err := svc.RegisterLearner(ctx, domain.LearnerRegisterInput{
		Username:    "xiaoming",
		Password:    "Study@123",
		DisplayName: "小明",
	})
	if err != nil {
		t.Fatalf("register learner: %v", err)
	}
	if user.Username != "xiaoming" {
		t.Fatalf("expected username xiaoming, got %q", user.Username)
	}

	authenticated, err := svc.AuthenticateLearner(ctx, "xiaoming", "Study@123")
	if err != nil {
		t.Fatalf("authenticate learner: %v", err)
	}
	if authenticated.DisplayName != "小明" {
		t.Fatalf("expected display name 小明, got %q", authenticated.DisplayName)
	}
}
