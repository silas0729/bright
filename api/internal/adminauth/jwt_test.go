package adminauth

import (
	"testing"
	"time"

	"brights/api/internal/domain"
)

func TestIssueAndParseAdminToken(t *testing.T) {
	manager := NewManager("brights-admin", "secret-value", 30*time.Minute)

	session, err := manager.IssueAdminSession(domain.AdminUser{
		ID:          1,
		Username:    "superadmin",
		DisplayName: "Root",
		Role:        "super_admin",
		IsSuper:     true,
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	claims, err := manager.ParseAdminToken(session.AccessToken)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.AdminID != 1 {
		t.Fatalf("expected admin id 1, got %d", claims.AdminID)
	}
	if claims.Username != "superadmin" {
		t.Fatalf("expected username superadmin, got %q", claims.Username)
	}
}
