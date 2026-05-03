package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

func (s *Server) handleAdminKnowledgeBaseDocuments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListKnowledgeBaseDocuments(c.Request.Context(), domain.KnowledgeBaseDocumentFilter{
		SubjectKey: c.Query("subject"),
		Query:      c.Query("q"),
		Page:       page,
		PageSize:   pageSize,
		IncludeAll: true,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminImportKnowledgeBase(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, domainError("please choose a txt, md, csv, or xlsx file"))
		return
	}

	subjectKey := defaultIfBlank(c.PostForm("subject_key"), "english")
	title := strings.TrimSpace(c.PostForm("title"))

	tempDir, err := os.MkdirTemp("", "brights-kb-*")
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	defer os.RemoveAll(tempDir)

	fileName := filepath.Base(strings.TrimSpace(fileHeader.Filename))
	if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
		fileName = "knowledge-base.txt"
	}
	tempPath := filepath.Join(tempDir, fileName)
	if err := c.SaveUploadedFile(fileHeader, tempPath); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.ImportKnowledgeBaseFromFile(c.Request.Context(), domain.ImportKnowledgeBaseInput{
		Path:       tempPath,
		SubjectKey: subjectKey,
		Title:      title,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	result.Document.SourceFileName = fileName
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleSearchKnowledgeBase(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	claims, hasClaims := s.optionalLearnerClaims(c)
	learnerID := uint(0)
	if hasClaims {
		learnerID = claims.UserID
	}
	result, err := s.service.SearchKnowledgeBase(c.Request.Context(), domain.SearchKnowledgeBaseInput{
		SubjectKey:    c.Query("subject"),
		Query:         firstNonEmpty(c.Query("q"), c.Query("query")),
		Page:          page,
		PageSize:      pageSize,
		LearnerUserID: learnerID,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleAdminUpdateKnowledgeBaseDocumentStatus(c *gin.Context) {
	documentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || documentID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid knowledge base document id"))
		return
	}

	var input domain.UpdateKnowledgeBaseDocumentStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateKnowledgeBaseDocumentStatus(c.Request.Context(), uint(documentID), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleAdminDeleteKnowledgeBaseDocument(c *gin.Context) {
	documentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || documentID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid knowledge base document id"))
		return
	}

	if err := s.service.DeleteKnowledgeBaseDocument(c.Request.Context(), uint(documentID)); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleLearnerKnowledgeBaseDocuments(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := s.service.ListKnowledgeBaseDocuments(c.Request.Context(), domain.KnowledgeBaseDocumentFilter{
		SubjectKey:         c.Query("subject"),
		Query:              c.Query("q"),
		Page:               page,
		PageSize:           pageSize,
		OnlyOwned:          true,
		OwnerLearnerUserID: claims.UserID,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerImportKnowledgeBase(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, domainError("please choose a txt, md, csv, or xlsx file"))
		return
	}

	subjectKey := defaultIfBlank(c.PostForm("subject_key"), "english")
	title := strings.TrimSpace(c.PostForm("title"))

	tempDir, err := os.MkdirTemp("", "brights-learner-kb-*")
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	defer os.RemoveAll(tempDir)

	fileName := filepath.Base(strings.TrimSpace(fileHeader.Filename))
	if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
		fileName = "knowledge-base.txt"
	}
	tempPath := filepath.Join(tempDir, fileName)
	if err := c.SaveUploadedFile(fileHeader, tempPath); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.service.ImportLearnerKnowledgeBaseFromFile(c.Request.Context(), claims.UserID, domain.ImportKnowledgeBaseInput{
		Path:       tempPath,
		SubjectKey: subjectKey,
		Title:      title,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	result.Document.SourceFileName = fileName
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLearnerUpdateKnowledgeBaseDocumentStatus(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	documentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || documentID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid knowledge base document id"))
		return
	}

	var input domain.UpdateKnowledgeBaseDocumentStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	item, err := s.service.UpdateLearnerKnowledgeBaseDocumentStatus(c.Request.Context(), claims.UserID, uint(documentID), input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleLearnerDeleteKnowledgeBaseDocument(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	documentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || documentID == 0 {
		writeError(c, http.StatusBadRequest, domainError("invalid knowledge base document id"))
		return
	}

	if err := s.service.DeleteLearnerKnowledgeBaseDocument(c.Request.Context(), claims.UserID, uint(documentID)); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
