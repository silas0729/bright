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

func TestListWordsAndClassificationsHideVIP(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	seedInputs := []domain.CreateWordInput{
		{SubjectKey: "english", Classification: "travel", Term: "boarding", Translation: "登机", IsVIP: false},
		{SubjectKey: "english", Classification: "travel", Term: "upgrade", Translation: "升舱", IsVIP: true},
		{SubjectKey: "english", Classification: "finance", Term: "invoice", Translation: "发票", IsVIP: true},
		{SubjectKey: "english", Term: "context", Translation: "语境", IsVIP: false},
	}
	for _, input := range seedInputs {
		if _, err := svc.CreateWord(ctx, input); err != nil {
			t.Fatalf("create word %q: %v", input.Term, err)
		}
	}

	freeWords, err := svc.ListWords(ctx, domain.WordFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   20,
		HideVIP:    true,
	})
	if err != nil {
		t.Fatalf("list free words: %v", err)
	}
	if freeWords.Total != 2 {
		t.Fatalf("expected 2 free words, got %d", freeWords.Total)
	}
	for _, item := range freeWords.Items {
		if item.IsVIP {
			t.Fatalf("expected hide vip filter to exclude vip words, got %+v", item)
		}
	}

	freeStats, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   10,
		HideVIP:    true,
	})
	if err != nil {
		t.Fatalf("list free classification stats: %v", err)
	}
	if freeStats.Total != 3 {
		t.Fatalf("expected 3 visible classifications for non-members, got %d", freeStats.Total)
	}

	freeStatsByName := make(map[string]domain.ClassificationStat, len(freeStats.Items))
	for _, item := range freeStats.Items {
		freeStatsByName[item.Name] = item
	}
	if freeStatsByName["travel"].Count != 2 || freeStatsByName["travel"].AccessibleCount != 1 || !freeStatsByName["travel"].HasMemberContent {
		t.Fatalf("unexpected travel stat for non-members: %+v", freeStatsByName["travel"])
	}
	if freeStatsByName["Unclassified"].Count != 1 || freeStatsByName["Unclassified"].AccessibleCount != 1 {
		t.Fatalf("unexpected unclassified stat for non-members: %+v", freeStatsByName["Unclassified"])
	}
	if freeStatsByName["finance"].Count != 1 || freeStatsByName["finance"].AccessibleCount != 0 || !freeStatsByName["finance"].RequiresMembership {
		t.Fatalf("unexpected finance stat for non-members: %+v", freeStatsByName["finance"])
	}

	allStats, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list all classification stats: %v", err)
	}

	allCounts := make(map[string]int, len(allStats.Items))
	for _, item := range allStats.Items {
		allCounts[item.Name] = item.Count
	}
	if allCounts["travel"] != 2 || allCounts["finance"] != 1 || allCounts["Unclassified"] != 1 {
		t.Fatalf("unexpected all classification counts: %+v", allCounts)
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

func TestBatchUpdateWordVIPByClassification(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	travelOne, err := svc.CreateWord(ctx, domain.CreateWordInput{
		SubjectKey:     "english",
		Classification: "travel",
		Term:           "boarding",
		Translation:    "登机",
	})
	if err != nil {
		t.Fatalf("create first travel word: %v", err)
	}
	travelTwo, err := svc.CreateWord(ctx, domain.CreateWordInput{
		SubjectKey:     "english",
		Classification: "travel",
		Term:           "luggage",
		Translation:    "行李",
	})
	if err != nil {
		t.Fatalf("create second travel word: %v", err)
	}
	otherWord, err := svc.CreateWord(ctx, domain.CreateWordInput{
		SubjectKey:     "english",
		Classification: "finance",
		Term:           "invoice",
		Translation:    "发票",
	})
	if err != nil {
		t.Fatalf("create finance word: %v", err)
	}

	if travelOne.CategoryID == nil || *travelOne.CategoryID == 0 {
		t.Fatal("expected travel category id")
	}

	result, err := svc.BatchUpdateWordVIP(ctx, domain.BatchUpdateWordVIPInput{
		SubjectKey: "english",
		CategoryID: travelOne.CategoryID,
		IsVIP:      true,
	})
	if err != nil {
		t.Fatalf("batch update vip: %v", err)
	}
	if result.UpdatedCount != 2 {
		t.Fatalf("expected 2 updated words, got %d", result.UpdatedCount)
	}
	if result.Classification != "travel" {
		t.Fatalf("expected classification travel, got %q", result.Classification)
	}

	var words []storage.Word
	if err := svc.db.WithContext(ctx).Order("id asc").Find(&words).Error; err != nil {
		t.Fatalf("load words: %v", err)
	}
	if len(words) != 3 {
		t.Fatalf("expected 3 words, got %d", len(words))
	}
	if !words[0].IsVIP || !words[1].IsVIP {
		t.Fatalf("expected travel words to become VIP, got %+v", words[:2])
	}
	if words[2].IsVIP {
		t.Fatalf("expected finance word to remain non-VIP, got %+v", words[2])
	}

	hiddenStats, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   10,
		HideVIP:    true,
	})
	if err != nil {
		t.Fatalf("list free classification stats after vip update: %v", err)
	}
	if hiddenStats.Total != 2 {
		t.Fatalf("expected two visible classifications after vip update, got %d", hiddenStats.Total)
	}
	hiddenStatsByName := make(map[string]domain.ClassificationStat, len(hiddenStats.Items))
	for _, item := range hiddenStats.Items {
		hiddenStatsByName[item.Name] = item
	}
	if hiddenStatsByName["finance"].AccessibleCount != 1 || hiddenStatsByName["finance"].Count != 1 {
		t.Fatalf("unexpected finance stat after vip update: %+v", hiddenStatsByName["finance"])
	}
	if hiddenStatsByName["travel"].AccessibleCount != 0 || !hiddenStatsByName["travel"].RequiresMembership {
		t.Fatalf("unexpected travel stat after vip update: %+v", hiddenStatsByName["travel"])
	}

	resetResult, err := svc.BatchUpdateWordVIP(ctx, domain.BatchUpdateWordVIPInput{
		SubjectKey:     "english",
		Classification: "travel",
		IsVIP:          false,
	})
	if err != nil {
		t.Fatalf("reset vip by classification: %v", err)
	}
	if resetResult.UpdatedCount != 2 {
		t.Fatalf("expected 2 reset words, got %d", resetResult.UpdatedCount)
	}

	var reloaded []storage.Word
	if err := svc.db.WithContext(ctx).Order("id asc").Find(&reloaded).Error; err != nil {
		t.Fatalf("reload all words: %v", err)
	}
	if len(reloaded) != 3 {
		t.Fatalf("expected 3 reloaded words, got %d", len(reloaded))
	}
	for _, item := range reloaded {
		if item.Term == travelTwo.Term && item.IsVIP {
			t.Fatal("expected travel word to be reset to non-VIP")
		}
		if item.Term == otherWord.Term && item.IsVIP {
			t.Fatal("expected finance word to remain non-VIP")
		}
	}

	visibleStats, err := svc.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
		SubjectKey: "english",
		Page:       1,
		PageSize:   10,
		HideVIP:    true,
	})
	if err != nil {
		t.Fatalf("list free classification stats after vip reset: %v", err)
	}

	visibleStatsByName := make(map[string]domain.ClassificationStat, len(visibleStats.Items))
	for _, item := range visibleStats.Items {
		visibleStatsByName[item.Name] = item
	}
	if visibleStatsByName["travel"].AccessibleCount != 2 || visibleStatsByName["finance"].AccessibleCount != 1 {
		t.Fatalf("unexpected free classification stats after vip reset: %+v", visibleStatsByName)
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
