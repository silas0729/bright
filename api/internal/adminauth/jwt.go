package adminauth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"brights/api/internal/domain"
)

type Manager struct {
	issuer    string
	secretKey []byte
	ttl       time.Duration
}

type Claims struct {
	AdminID    uint   `json:"admin_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	IsSuper    bool   `json:"is_super"`
	TokenUsage string `json:"token_usage"`
	jwt.RegisteredClaims
}

func NewManager(issuer, secret string, ttl time.Duration) *Manager {
	return &Manager{
		issuer:    issuer,
		secretKey: []byte(secret),
		ttl:       ttl,
	}
}

func (m *Manager) IssueAdminSession(admin domain.AdminUser) (domain.AdminSession, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		AdminID:    admin.ID,
		Username:   admin.Username,
		Role:       admin.Role,
		IsSuper:    admin.IsSuper,
		TokenUsage: "admin_access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   admin.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secretKey)
	if err != nil {
		return domain.AdminSession{}, err
	}

	return domain.AdminSession{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		Admin:       admin,
	}, nil
}

func (m *Manager) ParseAdminToken(rawToken string) (Claims, error) {
	if rawToken == "" {
		return Claims{}, errors.New("missing token")
	}

	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secretKey, nil
	}, jwt.WithIssuer(m.issuer))
	if err != nil {
		return Claims{}, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return Claims{}, errors.New("invalid token")
	}
	if claims.TokenUsage != "admin_access" {
		return Claims{}, errors.New("invalid token usage")
	}
	return *claims, nil
}
