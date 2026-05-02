package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"brights/api/internal/adminauth"
	"brights/api/internal/domain"
	"brights/api/internal/service"
	"brights/api/internal/userauth"
)

type Server struct {
	service  *service.Service
	auth     *adminauth.Manager
	userAuth *userauth.Manager
}

func NewServer(service *service.Service, auth *adminauth.Manager, userAuth *userauth.Manager) *Server {
	return &Server{service: service, auth: auth, userAuth: userAuth}
}

func (s *Server) Routes() http.Handler {
	if strings.TrimSpace(os.Getenv("GIN_MODE")) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	v1 := router.Group("/api/v1")
	v1.GET("/health", s.handleHealth)
	v1.GET("/subjects", s.handleSubjects)
	v1.GET("/stats", s.handleStats)
	v1.GET("/classifications", s.handleClassifications)
	v1.GET("/categories", s.handleCategories)
	v1.GET("/grades", s.handleGrades)
	v1.GET("/words", s.handleWords)
	v1.GET("/plans", s.handlePlans)
	v1.GET("/site/settings", s.handleSiteSettings)
	v1.GET("/auth/captcha", s.handleLearnerCaptcha)
	v1.POST("/auth/register", s.handleLearnerRegister)
	v1.POST("/auth/login", s.handleLearnerLogin)
	v1.POST("/payments/wechat/orders", s.handleCreateWechatOrder)
	v1.GET("/payments/wechat/orders/:orderNo", s.handleGetWechatOrderStatus)
	v1.POST("/payments/wechat/notify", s.handleWechatPayNotify)
	v1.GET("/admin/setup/status", s.handleAdminSetupStatus)
	v1.POST("/admin/setup/bootstrap", s.handleAdminSetupBootstrap)

	userProtected := v1.Group("/auth")
	userProtected.Use(s.userRequired())
	userProtected.GET("/me", s.handleLearnerMe)
	userProtected.POST("/logout", s.handleLearnerLogout)

	adminAuth := v1.Group("/admin/auth")
	adminAuth.GET("/captcha", s.handleAdminCaptcha)
	adminAuth.POST("/login", s.handleAdminLogin)

	adminProtected := v1.Group("/admin")
	adminProtected.Use(s.authRequired())
	adminProtected.GET("/auth/me", s.handleAdminMe)
	adminProtected.POST("/auth/refresh", s.handleAdminRefresh)
	adminProtected.POST("/auth/logout", s.handleAdminLogout)
	adminProtected.POST("/auth/change-password", s.handleAdminChangePassword)
	adminProtected.GET("/roles", s.permissionRequired("admin.read"), s.handleAdminRoles)
	adminProtected.GET("/users", s.permissionRequired("admin.read"), s.handleAdminUsers)
	adminProtected.GET("/learners", s.permissionRequired("learner.read"), s.handleAdminLearners)
	adminProtected.POST("/users", s.permissionRequired("admin.write"), s.handleCreateAdminUser)
	adminProtected.PUT("/users/:id", s.permissionRequired("admin.write"), s.handleUpdateAdminUser)
	adminProtected.PUT("/learners/:id", s.permissionRequired("learner.write"), s.handleUpdateLearnerUser)
	adminProtected.POST("/roles", s.permissionRequired("role.write"), s.handleCreateAdminRole)
	adminProtected.PUT("/roles/:id", s.permissionRequired("role.write"), s.handleUpdateAdminRole)
	adminProtected.GET("/site/settings", s.permissionRequired("site.read"), s.handleAdminSiteSettings)
	adminProtected.PUT("/site/settings", s.permissionRequired("site.write"), s.handleAdminSaveSiteSettings)
	adminProtected.GET("/words", s.permissionRequired("catalog.read"), s.handleAdminWords)
	adminProtected.GET("/categories", s.permissionRequired("catalog.read"), s.handleAdminCategories)
	adminProtected.GET("/grades", s.permissionRequired("grade.read"), s.handleAdminGrades)
	adminProtected.GET("/plans", s.permissionRequired("plan.read"), s.handleAdminPlans)
	adminProtected.GET("/wechatpay/config", s.permissionRequired("payment.read"), s.handleAdminGetWechatPayConfig)
	adminProtected.POST("/wechatpay/config", s.permissionRequired("payment.write"), s.handleAdminSaveWechatPayConfig)
	adminProtected.GET("/payments/orders", s.permissionRequired("payment.read"), s.handleAdminPaymentOrders)
	adminProtected.GET("/payments/orders/:orderNo", s.permissionRequired("payment.read"), s.handleAdminPaymentOrderDetail)
	adminProtected.GET("/payments/subscriptions", s.permissionRequired("payment.read"), s.handleAdminSubscriptions)
	adminProtected.GET("/payments/subscriptions/:id", s.permissionRequired("payment.read"), s.handleAdminSubscription)

	admin := v1.Group("/admin")
	admin.Use(s.authRequired())
	admin.POST("/import/local", s.permissionRequired("catalog.write"), s.handleImportLocal)
	admin.POST("/subjects", s.permissionRequired("subject.write"), s.handleCreateSubject)
	admin.PUT("/subjects/:id", s.permissionRequired("subject.write"), s.handleUpdateSubject)
	admin.POST("/categories", s.permissionRequired("catalog.write"), s.handleCreateCategory)
	admin.PUT("/categories/:id", s.permissionRequired("catalog.write"), s.handleUpdateCategory)
	admin.POST("/grades", s.permissionRequired("grade.write"), s.handleCreateGrade)
	admin.PUT("/grades/:id", s.permissionRequired("grade.write"), s.handleUpdateGrade)
	admin.POST("/words", s.permissionRequired("catalog.write"), s.handleCreateWord)
	admin.PUT("/words/:id", s.permissionRequired("catalog.write"), s.handleUpdateWord)
	admin.POST("/plans", s.permissionRequired("plan.write"), s.handleCreatePlan)
	admin.PUT("/plans/:id", s.permissionRequired("plan.write"), s.handleUpdatePlan)
	admin.DELETE("/plans/:id", s.permissionRequired("plan.write"), s.handleDeletePlan)
	admin.PUT("/payments/subscriptions/:id", s.permissionRequired("payment.write"), s.handleAdminUpdateSubscription)

	return router
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleSubjects(c *gin.Context) {
	items, err := s.service.ListSubjects(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleStats(c *gin.Context) {
	stats, err := s.service.Stats(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Server) handleClassifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "8"))
	items, err := s.service.ListClassificationStatsPaged(c.Request.Context(), domain.ClassificationStatFilter{
		SubjectKey: c.Query("subject"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleCategories(c *gin.Context) {
	items, err := s.service.ListCategories(
		c.Request.Context(),
		c.Query("subject"),
		defaultIfBlank(c.Query("kind"), "topic"),
	)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleGrades(c *gin.Context) {
	items, err := s.service.ListGrades(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleWords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	subjectID, _ := strconv.ParseUint(c.DefaultQuery("subject_id", "0"), 10, 64)
	categoryID, _ := strconv.ParseUint(c.DefaultQuery("category_id", "0"), 10, 64)
	gradeID, _ := strconv.ParseUint(c.DefaultQuery("grade_id", "0"), 10, 64)

	result, err := s.service.ListWords(c.Request.Context(), domain.WordFilter{
		SubjectID:      uint(subjectID),
		SubjectKey:     c.Query("subject"),
		CategoryID:     uint(categoryID),
		Classification: c.Query("classification"),
		GradeID:        uint(gradeID),
		Query:          c.Query("q"),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handlePlans(c *gin.Context) {
	items, err := s.service.ListPlans(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminLogin(c *gin.Context) {
	var input domain.AdminLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if err := s.service.VerifyCaptcha("admin_login", input.CaptchaID, input.CaptchaAnswer); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	admin, err := s.service.AuthenticateAdmin(c.Request.Context(), input.Username, input.Password)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	session, err := s.buildAdminSession(c, admin)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, session)
}

func (s *Server) handleAdminMe(c *gin.Context) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	admin, err := s.service.GetAdminByID(c.Request.Context(), claims.AdminID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	c.JSON(http.StatusOK, admin)
}

func (s *Server) handleAdminRefresh(c *gin.Context) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	admin, err := s.service.GetAdminByID(c.Request.Context(), claims.AdminID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	session, err := s.buildAdminSession(c, admin)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

func (s *Server) handleAdminLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleAdminChangePassword(c *gin.Context) {
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var input domain.ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if err := s.service.ChangeAdminPassword(c.Request.Context(), claims.AdminID, input.OldPassword, input.NewPassword); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleAdminRoles(c *gin.Context) {
	items, err := s.service.ListAdminRoles(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleAdminUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListAdminUsers(c.Request.Context(), domain.AdminUserFilter{
		Query:    c.Query("q"),
		Role:     c.Query("role"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleCreateAdminUser(c *gin.Context) {
	var input domain.CreateAdminUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.CreateAdminUser(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateAdminUser(c *gin.Context) {
	adminID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || adminID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid admin id"))
		return
	}
	claims, err := adminClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var input domain.UpdateAdminUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.UpdateAdminUser(c.Request.Context(), uint(adminID), claims.AdminID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleCreateAdminRole(c *gin.Context) {
	var input domain.CreateAdminRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.CreateAdminRole(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateAdminRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || roleID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid role id"))
		return
	}
	var input domain.UpdateAdminRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.UpdateAdminRole(c.Request.Context(), uint(roleID), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleAdminWords(c *gin.Context) {
	s.handleWords(c)
}

func (s *Server) handleAdminCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListCategoriesPaged(c.Request.Context(), domain.CategoryFilter{
		SubjectKey: c.Query("subject"),
		Kind:       defaultIfBlank(c.Query("kind"), "topic"),
		Query:      c.Query("q"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminGrades(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListGradesPaged(c.Request.Context(), domain.GradeFilter{
		Stage:    c.Query("stage"),
		Query:    c.Query("q"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleImportLocal(c *gin.Context) {
	if strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			writeError(c, http.StatusBadRequest, domainError("please choose a csv or xlsx file"))
			return
		}

		replace := true
		if rawReplace := strings.TrimSpace(c.PostForm("replace")); rawReplace != "" {
			parsed, parseErr := strconv.ParseBool(rawReplace)
			if parseErr != nil {
				writeError(c, http.StatusBadRequest, domainError("replace must be true or false"))
				return
			}
			replace = parsed
		}

		subjectKey := defaultIfBlank(c.PostForm("subject_key"), "english")
		tempDir, err := os.MkdirTemp("", "brights-import-*")
		if err != nil {
			writeError(c, http.StatusInternalServerError, err)
			return
		}
		defer os.RemoveAll(tempDir)

		fileName := filepath.Base(strings.TrimSpace(fileHeader.Filename))
		if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
			fileName = "upload.csv"
		}
		tempPath := filepath.Join(tempDir, fileName)
		if err := c.SaveUploadedFile(fileHeader, tempPath); err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}

		replacePtr := replace
		result, err := s.service.ImportWordsFromFile(c.Request.Context(), domain.ImportWordsInput{
			Path:       tempPath,
			SubjectKey: subjectKey,
			Replace:    &replacePtr,
		})
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
		result.Path = fileName
		c.JSON(http.StatusOK, result)
		return
	}

	var input domain.ImportWordsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.SubjectKey) == "" {
		input.SubjectKey = "english"
	}
	result, err := s.service.ImportWordsFromFile(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleCreateSubject(c *gin.Context) {
	var input domain.CreateSubjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.CreateSubject(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateSubject(c *gin.Context) {
	subjectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || subjectID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid subject id"))
		return
	}
	var input domain.UpdateSubjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.UpdateSubject(c.Request.Context(), uint(subjectID), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleCreateCategory(c *gin.Context) {
	var input domain.CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.CreateCategory(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateCategory(c *gin.Context) {
	categoryID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || categoryID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid category id"))
		return
	}
	var input domain.UpdateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.UpdateCategory(c.Request.Context(), uint(categoryID), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleCreateGrade(c *gin.Context) {
	var input domain.CreateGradeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.CreateGrade(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateGrade(c *gin.Context) {
	gradeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || gradeID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid grade id"))
		return
	}
	var input domain.UpdateGradeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.UpdateGrade(c.Request.Context(), uint(gradeID), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleCreateWord(c *gin.Context) {
	var input domain.CreateWordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.CreateWord(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleUpdateWord(c *gin.Context) {
	wordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || wordID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid word id"))
		return
	}
	var input domain.UpdateWordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.UpdateWord(c.Request.Context(), wordID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleCreatePlan(c *gin.Context) {
	var input domain.CreatePlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.CreatePlan(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func writeError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

func domainError(message string) error {
	return &httpError{message: message}
}

type httpError struct {
	message string
}

func (e *httpError) Error() string {
	return e.message
}

func defaultIfBlank(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
