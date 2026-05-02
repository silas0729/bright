package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"brights/api/internal/domain"
)

const (
	captchaSceneLearnerRegister = "learner_register"
	captchaSceneLearnerLogin    = "learner_login"
	captchaSceneAdminLogin      = "admin_login"
	captchaTTL                  = 5 * time.Minute
)

type captchaEntry struct {
	answer    string
	expiresAt time.Time
}

func (s *Service) IssueCaptcha(scene string) (domain.CaptchaChallenge, error) {
	normalizedScene := normalizeCaptchaScene(scene)
	captchaID := randomCaptchaToken(24)
	answer := randomCaptchaAnswer(5)
	expiresAt := time.Now().Add(captchaTTL)

	s.captchaMu.Lock()
	defer s.captchaMu.Unlock()

	s.cleanupExpiredCaptchasLocked(time.Now())
	s.captchas[captchaStoreKey(normalizedScene, captchaID)] = captchaEntry{
		answer:    strings.ToLower(answer),
		expiresAt: expiresAt,
	}

	return domain.CaptchaChallenge{
		Scene:     normalizedScene,
		CaptchaID: captchaID,
		ImageData: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(buildCaptchaSVG(answer))),
		ExpiresIn: int64(captchaTTL.Seconds()),
	}, nil
}

func (s *Service) VerifyCaptcha(scene, captchaID, captchaAnswer string) error {
	captchaID = strings.TrimSpace(captchaID)
	captchaAnswer = strings.TrimSpace(captchaAnswer)
	if captchaID == "" || captchaAnswer == "" {
		return errors.New("captcha is required")
	}

	normalizedScene := normalizeCaptchaScene(scene)
	key := captchaStoreKey(normalizedScene, captchaID)
	now := time.Now()

	s.captchaMu.Lock()
	defer s.captchaMu.Unlock()

	s.cleanupExpiredCaptchasLocked(now)

	entry, ok := s.captchas[key]
	if !ok || now.After(entry.expiresAt) {
		delete(s.captchas, key)
		return errors.New("captcha expired or invalid")
	}
	if !strings.EqualFold(captchaAnswer, entry.answer) {
		return errors.New("captcha mismatch")
	}

	delete(s.captchas, key)
	return nil
}

func normalizeCaptchaScene(scene string) string {
	switch strings.ToLower(strings.TrimSpace(scene)) {
	case captchaSceneLearnerRegister:
		return captchaSceneLearnerRegister
	case captchaSceneAdminLogin:
		return captchaSceneAdminLogin
	case "", captchaSceneLearnerLogin:
		return captchaSceneLearnerLogin
	default:
		return captchaSceneLearnerLogin
	}
}

func captchaStoreKey(scene, captchaID string) string {
	return scene + ":" + captchaID
}

func (s *Service) cleanupExpiredCaptchasLocked(now time.Time) {
	for key, entry := range s.captchas {
		if now.After(entry.expiresAt) {
			delete(s.captchas, key)
		}
	}
}

func randomCaptchaToken(length int) string {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	return randomFromAlphabet(alphabet, length)
}

func randomCaptchaAnswer(length int) string {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	return randomFromAlphabet(alphabet, length)
}

func randomFromAlphabet(alphabet string, length int) string {
	if length <= 0 {
		return ""
	}
	builder := strings.Builder{}
	builder.Grow(length)
	for i := 0; i < length; i++ {
		builder.WriteByte(alphabet[randomInt(len(alphabet))])
	}
	return builder.String()
}

func buildCaptchaSVG(answer string) string {
	const width = 160
	const height = 56
	const baseline = 36

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height))
	builder.WriteString(`<rect width="100%" height="100%" rx="12" fill="#F8FAFC"/>`)
	for i := 0; i < 6; i++ {
		builder.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1.4" opacity="0.55"/>`, randomInt(width), randomInt(height), randomInt(width), randomInt(height), randomSoftColor()))
	}
	for i, char := range answer {
		x := 18 + i*26 + randomInt(6)
		y := baseline + randomInt(8)
		rotation := randomInt(25) - 12
		fontSize := 26 + randomInt(6)
		builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-size="%d" font-family="Arial, sans-serif" font-weight="700" fill="%s" transform="rotate(%d %d %d)">%c</text>`, x, y, fontSize, randomDarkColor(), rotation, x, y, char))
	}
	for i := 0; i < 18; i++ {
		builder.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="%s" opacity="0.35"/>`, randomInt(width), randomInt(height), 1+randomInt(2), randomSoftColor()))
	}
	builder.WriteString(`</svg>`)
	return builder.String()
}

func randomSoftColor() string {
	return fmt.Sprintf("rgb(%d,%d,%d)", 120+randomInt(100), 120+randomInt(100), 120+randomInt(100))
}

func randomDarkColor() string {
	return fmt.Sprintf("rgb(%d,%d,%d)", randomInt(110), randomInt(110), randomInt(110))
}

func randomInt(max int) int {
	if max <= 1 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}
