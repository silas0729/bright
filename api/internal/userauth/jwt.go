package userauth

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
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
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

func (m *Manager) IssueSession(user domain.LearnerUser) (domain.LearnerSession, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		UserID:     user.ID,
		Username:   user.Username,
		TokenUsage: "user_access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secretKey)
	if err != nil {
		return domain.LearnerSession{}, err
	}

	return domain.LearnerSession{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user,
	}, nil
}

func (m *Manager) ParseToken(rawToken string) (Claims, error) {
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
	if claims.TokenUsage != "user_access" {
		return Claims{}, errors.New("invalid token usage")
	}
	return *claims, nil
}
