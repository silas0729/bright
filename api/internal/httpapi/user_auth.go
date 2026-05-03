package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
	"brights/api/internal/userauth"
)

const userClaimsContextKey = "user_claims"

func (s *Server) userRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			writeError(c, http.StatusUnauthorized, errors.New("missing authorization header"))
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(c, http.StatusUnauthorized, errors.New("invalid authorization header"))
			c.Abort()
			return
		}

		claims, err := s.userAuth.ParseToken(strings.TrimSpace(parts[1]))
		if err != nil {
			writeError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		c.Set(userClaimsContextKey, claims)
		c.Next()
	}
}

func learnerClaimsFromContext(c *gin.Context) (userauth.Claims, error) {
	value, ok := c.Get(userClaimsContextKey)
	if !ok {
		return userauth.Claims{}, errors.New("missing user claims")
	}
	claims, ok := value.(userauth.Claims)
	if !ok {
		return userauth.Claims{}, errors.New("invalid user claims")
	}
	return claims, nil
}

func (s *Server) optionalLearnerClaims(c *gin.Context) (userauth.Claims, bool) {
	if s.userAuth == nil {
		return userauth.Claims{}, false
	}

	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return userauth.Claims{}, false
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return userauth.Claims{}, false
	}

	claims, err := s.userAuth.ParseToken(strings.TrimSpace(parts[1]))
	if err != nil {
		return userauth.Claims{}, false
	}
	return claims, true
}

func (s *Server) handleLearnerRegister(c *gin.Context) {
	var input domain.LearnerRegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if err := s.service.VerifyCaptcha("learner_register", input.CaptchaID, input.CaptchaAnswer); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	user, err := s.service.RegisterLearner(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	session, err := s.userAuth.IssueSession(user)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, session)
}

func (s *Server) handleLearnerLogin(c *gin.Context) {
	var input domain.LearnerLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if err := s.service.VerifyCaptcha("learner_login", input.CaptchaID, input.CaptchaAnswer); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	user, err := s.service.AuthenticateLearner(c.Request.Context(), input.Username, input.Password)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	session, err := s.userAuth.IssueSession(user)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

func (s *Server) handleLearnerMe(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	user, err := s.service.GetLearnerByIDWithMembership(c.Request.Context(), claims.UserID, c.Query("subject"))
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) handleLearnerLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}
