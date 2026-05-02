package service

import (
	"context"
	"testing"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	db, err := storage.Open("sqlite", "file::memory:?cache=shared", false)
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
