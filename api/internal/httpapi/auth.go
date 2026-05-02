package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/adminauth"
	"brights/api/internal/domain"
)

const adminClaimsContextKey = "admin_claims"

func (s *Server) authRequired() gin.HandlerFunc {
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

		claims, err := s.auth.ParseAdminToken(strings.TrimSpace(parts[1]))
		if err != nil {
			writeError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		c.Set(adminClaimsContextKey, claims)
		c.Next()
	}
}

func (s *Server) permissionRequired(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := adminClaimsFromContext(c)
		if err != nil {
			writeError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}
		if claims.IsSuper {
			c.Next()
			return
		}

		allowed, err := s.service.RoleHasPermission(c.Request.Context(), claims.Role, permission)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err)
			c.Abort()
			return
		}
		if !allowed {
			writeError(c, http.StatusForbidden, errors.New("permission denied"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func requireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := adminClaimsFromContext(c)
		if err != nil {
			writeError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}
		if !claims.IsSuper {
			writeError(c, http.StatusForbidden, errors.New("super admin permission required"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func adminClaimsFromContext(c *gin.Context) (adminauth.Claims, error) {
	value, ok := c.Get(adminClaimsContextKey)
	if !ok {
		return adminauth.Claims{}, errors.New("missing admin claims")
	}
	claims, ok := value.(adminauth.Claims)
	if !ok {
		return adminauth.Claims{}, errors.New("invalid admin claims")
	}
	return claims, nil
}

func (s *Server) buildAdminSession(c *gin.Context, admin domain.AdminUser) (domain.AdminSession, error) {
	current, err := s.service.GetAdminByID(c.Request.Context(), admin.ID)
	if err == nil {
		admin = current
	}
	return s.auth.IssueAdminSession(admin)
}
