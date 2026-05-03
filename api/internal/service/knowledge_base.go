package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"brights/api/internal/catalog"
	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

func (s *Service) ImportKnowledgeBaseFromFile(ctx context.Context, input domain.ImportKnowledgeBaseInput) (domain.ImportKnowledgeBaseResult, error) {
	return s.importKnowledgeBaseFromFile(ctx, input, 0)
}

func (s *Service) ImportLearnerKnowledgeBaseFromFile(ctx context.Context, learnerID uint, input domain.ImportKnowledgeBaseInput) (domain.ImportKnowledgeBaseResult, error) {
	if learnerID == 0 {
		return domain.ImportKnowledgeBaseResult{}, errors.New("learner id is required")
	}
	return s.importKnowledgeBaseFromFile(ctx, input, learnerID)
}

func (s *Service) importKnowledgeBaseFromFile(ctx context.Context, input domain.ImportKnowledgeBaseInput, ownerLearnerUserID uint) (domain.ImportKnowledgeBaseResult, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return domain.ImportKnowledgeBaseResult{}, errors.New("path is required")
	}

	subjectKey := normalizeKey(input.SubjectKey)
	if subjectKey == "" {
		subjectKey = "english"
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = strings.TrimSpace(filepath.Base(path))
	}

	if _, err := s.ensureSubject(ctx, subjectKey); err != nil {
		return domain.ImportKnowledgeBaseResult{}, err
	}

	sourceType := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if sourceType == "" {
		sourceType = "text"
	}

	rawChunks, err := catalog.LoadKnowledgeBaseChunksFromFile(path, title)
	if err != nil {
		return domain.ImportKnowledgeBaseResult{}, err
	}
	if len(rawChunks) == 0 {
		return domain.ImportKnowledgeBaseResult{}, errors.New("knowledge base file is empty")
	}

	documentModel := storage.KnowledgeBaseDocument{
		SubjectKey:     subjectKey,
		Title:          title,
		SourceFileName: filepath.Base(path),
		SourceType:     sourceType,
		Status:         "active",
		Visibility:     knowledgeBaseVisibilityPublic,
	}
	if ownerLearnerUserID > 0 {
		documentModel.Visibility = knowledgeBaseVisibilityPrivate
		documentModel.OwnerLearnerUserID = uintPtr(ownerLearnerUserID)
	}

	totalCharacters := 0
	chunks := make([]storage.KnowledgeBaseChunk, 0, len(rawChunks))
	for index, item := range rawChunks {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		chunkTitle := strings.TrimSpace(item.Title)
		if chunkTitle == "" {
			chunkTitle = title
		}
		characterCount := len([]rune(content))
		totalCharacters += characterCount
		chunks = append(chunks, storage.KnowledgeBaseChunk{
			SubjectKey:     subjectKey,
			Title:          chunkTitle,
			ChunkIndex:     index + 1,
			Content:        content,
			ContentSearch:  strings.ToLower(content),
			CharacterCount: characterCount,
		})
	}
	if len(chunks) == 0 {
		return domain.ImportKnowledgeBaseResult{}, errors.New("knowledge base file is empty")
	}

	documentModel.ChunkCount = len(chunks)
	documentModel.CharacterCount = totalCharacters

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&documentModel).Error; err != nil {
			return err
		}
		for index := range chunks {
			chunks[index].DocumentID = documentModel.ID
		}
		return tx.CreateInBatches(chunks, 200).Error
	}); err != nil {
		return domain.ImportKnowledgeBaseResult{}, err
	}

	return domain.ImportKnowledgeBaseResult{
		Document:       toKnowledgeBaseDocument(documentModel, ""),
		ChunkCount:     len(chunks),
		CharacterCount: totalCharacters,
	}, nil
}

func (s *Service) ListKnowledgeBaseDocuments(ctx context.Context, filter domain.KnowledgeBaseDocumentFilter) (domain.PagedKnowledgeBaseDocuments, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)

	query := s.db.WithContext(ctx).Model(&storage.KnowledgeBaseDocument{})
	if subjectKey := normalizeKey(filter.SubjectKey); subjectKey != "" {
		query = query.Where("subject_key = ?", subjectKey)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("title LIKE ? OR source_file_name LIKE ?", like, like)
	}
	switch {
	case filter.IncludeAll:
		// Admin view sees both public and learner-owned documents.
	case filter.OnlyOwned:
		if filter.OwnerLearnerUserID == 0 {
			return domain.PagedKnowledgeBaseDocuments{
				Items:    []domain.KnowledgeBaseDocument{},
				Total:    0,
				Page:     page,
				PageSize: pageSize,
			}, nil
		}
		query = query.Where("owner_learner_user_id = ?", filter.OwnerLearnerUserID)
	default:
		query = query.Where("owner_learner_user_id IS NULL")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedKnowledgeBaseDocuments{}, err
	}

	var models []storage.KnowledgeBaseDocument
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedKnowledgeBaseDocuments{}, err
	}

	ownerMap, err := s.knowledgeBaseOwnerMap(ctx, models)
	if err != nil {
		return domain.PagedKnowledgeBaseDocuments{}, err
	}

	items := make([]domain.KnowledgeBaseDocument, 0, len(models))
	for _, model := range models {
		items = append(items, toKnowledgeBaseDocument(model, ownerMap[ownerIDValue(model.OwnerLearnerUserID)]))
	}

	return domain.PagedKnowledgeBaseDocuments{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) SearchKnowledgeBase(ctx context.Context, input domain.SearchKnowledgeBaseInput) (domain.PagedKnowledgeBaseChunks, error) {
	page, pageSize := normalizePage(input.Page, input.PageSize, 10)

	queryText := strings.TrimSpace(input.Query)
	if queryText == "" {
		return domain.PagedKnowledgeBaseChunks{
			Items:    []domain.KnowledgeBaseChunk{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	query := s.db.WithContext(ctx).
		Model(&storage.KnowledgeBaseChunk{}).
		Joins("JOIN knowledge_base_documents ON knowledge_base_documents.id = knowledge_base_chunks.document_id").
		Where("knowledge_base_documents.status = ?", "active")
	if subjectKey := normalizeKey(input.SubjectKey); subjectKey != "" {
		query = query.Where("knowledge_base_chunks.subject_key = ?", subjectKey)
	}
	if input.LearnerUserID > 0 {
		query = query.Where(
			"(knowledge_base_documents.owner_learner_user_id IS NULL OR knowledge_base_documents.owner_learner_user_id = ?)",
			input.LearnerUserID,
		)
	} else {
		query = query.Where("knowledge_base_documents.owner_learner_user_id IS NULL")
	}

	lowerQuery := strings.ToLower(queryText)
	like := "%" + lowerQuery + "%"
	query = query.Where(
		"LOWER(knowledge_base_documents.title) LIKE ? OR LOWER(knowledge_base_chunks.title) LIKE ? OR content_search LIKE ? OR LOWER(content) LIKE ?",
		like,
		like,
		like,
		like,
	)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedKnowledgeBaseChunks{}, err
	}

	var models []storage.KnowledgeBaseChunk
	if err := query.Order("document_id desc, chunk_index asc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedKnowledgeBaseChunks{}, err
	}

	documentMap, err := s.knowledgeBaseDocumentMap(ctx, models)
	if err != nil {
		return domain.PagedKnowledgeBaseChunks{}, err
	}

	items := make([]domain.KnowledgeBaseChunk, 0, len(models))
	for _, model := range models {
		items = append(items, toKnowledgeBaseChunk(model, documentMap[model.DocumentID], queryText))
	}

	return domain.PagedKnowledgeBaseChunks{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) UpdateKnowledgeBaseDocumentStatus(ctx context.Context, id uint, input domain.UpdateKnowledgeBaseDocumentStatusInput) (domain.KnowledgeBaseDocument, error) {
	return s.updateKnowledgeBaseDocumentStatus(ctx, id, input, nil)
}

func (s *Service) UpdateLearnerKnowledgeBaseDocumentStatus(ctx context.Context, learnerID uint, id uint, input domain.UpdateKnowledgeBaseDocumentStatusInput) (domain.KnowledgeBaseDocument, error) {
	if learnerID == 0 {
		return domain.KnowledgeBaseDocument{}, errors.New("learner id is required")
	}
	return s.updateKnowledgeBaseDocumentStatus(ctx, id, input, uintPtr(learnerID))
}

func (s *Service) updateKnowledgeBaseDocumentStatus(ctx context.Context, id uint, input domain.UpdateKnowledgeBaseDocumentStatusInput, learnerID *uint) (domain.KnowledgeBaseDocument, error) {
	if id == 0 {
		return domain.KnowledgeBaseDocument{}, errors.New("document id is required")
	}

	status, err := normalizeKnowledgeBaseDocumentStatus(input.Status)
	if err != nil {
		return domain.KnowledgeBaseDocument{}, err
	}

	model, err := s.findKnowledgeBaseDocumentForActor(s.db.WithContext(ctx), id, learnerID)
	if err != nil {
		return domain.KnowledgeBaseDocument{}, err
	}

	if err := s.db.WithContext(ctx).Model(&model).Update("status", status).Error; err != nil {
		return domain.KnowledgeBaseDocument{}, err
	}
	model.Status = status

	ownerMap, err := s.knowledgeBaseOwnerMap(ctx, []storage.KnowledgeBaseDocument{model})
	if err != nil {
		return domain.KnowledgeBaseDocument{}, err
	}
	return toKnowledgeBaseDocument(model, ownerMap[ownerIDValue(model.OwnerLearnerUserID)]), nil
}

func (s *Service) DeleteKnowledgeBaseDocument(ctx context.Context, id uint) error {
	return s.deleteKnowledgeBaseDocument(ctx, id, nil)
}

func (s *Service) DeleteLearnerKnowledgeBaseDocument(ctx context.Context, learnerID uint, id uint) error {
	if learnerID == 0 {
		return errors.New("learner id is required")
	}
	return s.deleteKnowledgeBaseDocument(ctx, id, uintPtr(learnerID))
}

func (s *Service) deleteKnowledgeBaseDocument(ctx context.Context, id uint, learnerID *uint) error {
	if id == 0 {
		return errors.New("document id is required")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model, err := s.findKnowledgeBaseDocumentForActor(tx, id, learnerID)
		if err != nil {
			return err
		}
		if err := tx.Where("document_id = ?", id).Delete(&storage.KnowledgeBaseChunk{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model).Error
	})
}

func toKnowledgeBaseDocument(model storage.KnowledgeBaseDocument, ownerUsername string) domain.KnowledgeBaseDocument {
	return domain.KnowledgeBaseDocument{
		ID:                 model.ID,
		SubjectKey:         model.SubjectKey,
		Title:              model.Title,
		SourceFileName:     model.SourceFileName,
		SourceType:         model.SourceType,
		Status:             model.Status,
		Visibility:         normalizeKnowledgeBaseVisibility(model.Visibility),
		OwnerLearnerUserID: model.OwnerLearnerUserID,
		OwnerUsername:      strings.TrimSpace(ownerUsername),
		ChunkCount:         model.ChunkCount,
		CharacterCount:     model.CharacterCount,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func toKnowledgeBaseChunk(model storage.KnowledgeBaseChunk, document storage.KnowledgeBaseDocument, queryText string) domain.KnowledgeBaseChunk {
	documentTitle := strings.TrimSpace(document.Title)
	if documentTitle == "" {
		documentTitle = model.Title
	}
	snippetSource := model.Content
	if !strings.Contains(strings.ToLower(model.Content), strings.ToLower(strings.TrimSpace(queryText))) && documentTitle != "" {
		snippetSource = documentTitle
	}
	snippet, highlightedSnippet := buildKnowledgeBaseSnippet(snippetSource, queryText)

	return domain.KnowledgeBaseChunk{
		ID:                 model.ID,
		DocumentID:         model.DocumentID,
		SubjectKey:         model.SubjectKey,
		Title:              model.Title,
		DocumentTitle:      documentTitle,
		SourceFileName:     document.SourceFileName,
		SourceType:         document.SourceType,
		Status:             document.Status,
		ChunkIndex:         model.ChunkIndex,
		Content:            model.Content,
		Snippet:            snippet,
		HighlightedSnippet: highlightedSnippet,
		CharacterCount:     model.CharacterCount,
		CreatedAt:          model.CreatedAt,
	}
}

func (s *Service) knowledgeBaseDocumentMap(ctx context.Context, chunks []storage.KnowledgeBaseChunk) (map[uint]storage.KnowledgeBaseDocument, error) {
	result := make(map[uint]storage.KnowledgeBaseDocument)
	if len(chunks) == 0 {
		return result, nil
	}

	ids := make([]uint, 0, len(chunks))
	seen := make(map[uint]struct{}, len(chunks))
	for _, chunk := range chunks {
		if _, ok := seen[chunk.DocumentID]; ok || chunk.DocumentID == 0 {
			continue
		}
		seen[chunk.DocumentID] = struct{}{}
		ids = append(ids, chunk.DocumentID)
	}

	var documents []storage.KnowledgeBaseDocument
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&documents).Error; err != nil {
		return nil, err
	}
	for _, document := range documents {
		result[document.ID] = document
	}
	return result, nil
}

func (s *Service) knowledgeBaseOwnerMap(ctx context.Context, documents []storage.KnowledgeBaseDocument) (map[uint]string, error) {
	result := make(map[uint]string)
	if len(documents) == 0 {
		return result, nil
	}

	ownerIDs := make([]uint, 0, len(documents))
	seen := make(map[uint]struct{}, len(documents))
	for _, document := range documents {
		ownerID := ownerIDValue(document.OwnerLearnerUserID)
		if ownerID == 0 {
			continue
		}
		if _, ok := seen[ownerID]; ok {
			continue
		}
		seen[ownerID] = struct{}{}
		ownerIDs = append(ownerIDs, ownerID)
	}
	if len(ownerIDs) == 0 {
		return result, nil
	}

	var learners []storage.LearnerUser
	if err := s.db.WithContext(ctx).
		Select("id", "username").
		Where("id IN ?", ownerIDs).
		Find(&learners).Error; err != nil {
		return nil, err
	}
	for _, learner := range learners {
		result[learner.ID] = learner.Username
	}
	return result, nil
}

func (s *Service) findKnowledgeBaseDocumentForActor(db *gorm.DB, id uint, learnerID *uint) (storage.KnowledgeBaseDocument, error) {
	var model storage.KnowledgeBaseDocument
	query := db.Where("id = ?", id)
	if learnerID != nil {
		query = query.Where("owner_learner_user_id = ?", *learnerID)
	}
	if err := query.First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return storage.KnowledgeBaseDocument{}, errors.New("knowledge base document does not exist")
		}
		return storage.KnowledgeBaseDocument{}, err
	}
	return model, nil
}

func normalizeKnowledgeBaseDocumentStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "active":
		return "active", nil
	case "disabled":
		return "disabled", nil
	default:
		return "", errors.New("knowledge base document status must be active or disabled")
	}
}

const (
	knowledgeBaseVisibilityPublic  = "public"
	knowledgeBaseVisibilityPrivate = "private"
)

func normalizeKnowledgeBaseVisibility(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case knowledgeBaseVisibilityPrivate:
		return knowledgeBaseVisibilityPrivate
	default:
		return knowledgeBaseVisibilityPublic
	}
}

func buildKnowledgeBaseSnippet(content string, queryText string) (string, string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", ""
	}

	queryText = strings.TrimSpace(queryText)
	runes := []rune(content)
	if len(runes) <= 180 {
		return content, highlightFirstKnowledgeBaseMatch(content, queryText)
	}

	matchStart, matchLength := firstCaseInsensitiveRuneMatch(content, queryText)
	if matchStart < 0 {
		snippet := string(runes[:180]) + "..."
		return snippet, snippet
	}

	start := matchStart - 40
	if start < 0 {
		start = 0
	}
	end := matchStart + matchLength + 80
	if end > len(runes) {
		end = len(runes)
	}

	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet, highlightFirstKnowledgeBaseMatch(snippet, queryText)
}

func highlightFirstKnowledgeBaseMatch(content string, queryText string) string {
	content = strings.TrimSpace(content)
	queryText = strings.TrimSpace(queryText)
	if content == "" || queryText == "" {
		return content
	}

	matchStart, matchLength := firstCaseInsensitiveRuneMatch(content, queryText)
	if matchStart < 0 || matchLength <= 0 {
		return content
	}

	runes := []rune(content)
	return string(runes[:matchStart]) + "<<" + string(runes[matchStart:matchStart+matchLength]) + ">>" + string(runes[matchStart+matchLength:])
}

func firstCaseInsensitiveRuneMatch(content string, queryText string) (int, int) {
	contentRunes := []rune(content)
	queryRunes := []rune(strings.ToLower(queryText))
	lowerContentRunes := []rune(strings.ToLower(content))
	if len(queryRunes) == 0 || len(contentRunes) == 0 || len(queryRunes) > len(contentRunes) {
		return -1, 0
	}

	for index := 0; index <= len(lowerContentRunes)-len(queryRunes); index++ {
		if string(lowerContentRunes[index:index+len(queryRunes)]) == string(queryRunes) {
			return index, len(queryRunes)
		}
	}
	return -1, 0
}

func ownerIDValue(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}
