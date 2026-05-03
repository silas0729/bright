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
		Document:       toKnowledgeBaseDocument(documentModel),
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

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedKnowledgeBaseDocuments{}, err
	}

	var models []storage.KnowledgeBaseDocument
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return domain.PagedKnowledgeBaseDocuments{}, err
	}

	items := make([]domain.KnowledgeBaseDocument, 0, len(models))
	for _, model := range models {
		items = append(items, toKnowledgeBaseDocument(model))
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

	query := s.db.WithContext(ctx).Model(&storage.KnowledgeBaseChunk{})
	if subjectKey := normalizeKey(input.SubjectKey); subjectKey != "" {
		query = query.Where("subject_key = ?", subjectKey)
	}

	lowerQuery := strings.ToLower(queryText)
	like := "%" + lowerQuery + "%"
	query = query.Where(
		"LOWER(title) LIKE ? OR content_search LIKE ? OR LOWER(content) LIKE ?",
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

	items := make([]domain.KnowledgeBaseChunk, 0, len(models))
	for _, model := range models {
		items = append(items, toKnowledgeBaseChunk(model))
	}

	return domain.PagedKnowledgeBaseChunks{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func toKnowledgeBaseDocument(model storage.KnowledgeBaseDocument) domain.KnowledgeBaseDocument {
	return domain.KnowledgeBaseDocument{
		ID:             model.ID,
		SubjectKey:     model.SubjectKey,
		Title:          model.Title,
		SourceFileName: model.SourceFileName,
		SourceType:     model.SourceType,
		Status:         model.Status,
		ChunkCount:     model.ChunkCount,
		CharacterCount: model.CharacterCount,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}

func toKnowledgeBaseChunk(model storage.KnowledgeBaseChunk) domain.KnowledgeBaseChunk {
	return domain.KnowledgeBaseChunk{
		ID:             model.ID,
		DocumentID:     model.DocumentID,
		SubjectKey:     model.SubjectKey,
		Title:          model.Title,
		ChunkIndex:     model.ChunkIndex,
		Content:        model.Content,
		CharacterCount: model.CharacterCount,
		CreatedAt:      model.CreatedAt,
	}
}
