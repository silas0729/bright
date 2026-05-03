package domain

import "time"

type KnowledgeBaseDocument struct {
	ID                 uint      `json:"id"`
	SubjectKey         string    `json:"subject_key"`
	Title              string    `json:"title"`
	SourceFileName     string    `json:"source_file_name"`
	SourceType         string    `json:"source_type"`
	Status             string    `json:"status"`
	Visibility         string    `json:"visibility,omitempty"`
	OwnerLearnerUserID *uint     `json:"owner_learner_user_id,omitempty"`
	OwnerUsername      string    `json:"owner_username,omitempty"`
	ChunkCount         int       `json:"chunk_count"`
	CharacterCount     int       `json:"character_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type KnowledgeBaseChunk struct {
	ID                 uint      `json:"id"`
	DocumentID         uint      `json:"document_id"`
	SubjectKey         string    `json:"subject_key"`
	Title              string    `json:"title"`
	DocumentTitle      string    `json:"document_title,omitempty"`
	SourceFileName     string    `json:"source_file_name,omitempty"`
	SourceType         string    `json:"source_type,omitempty"`
	Status             string    `json:"status,omitempty"`
	ChunkIndex         int       `json:"chunk_index"`
	Content            string    `json:"content"`
	Snippet            string    `json:"snippet,omitempty"`
	HighlightedSnippet string    `json:"highlighted_snippet,omitempty"`
	CharacterCount     int       `json:"character_count"`
	CreatedAt          time.Time `json:"created_at"`
}

type KnowledgeBaseDocumentFilter struct {
	SubjectKey         string
	Query              string
	Page               int
	PageSize           int
	OnlyOwned          bool
	OwnerLearnerUserID uint
	IncludeAll         bool
}

type SearchKnowledgeBaseInput struct {
	SubjectKey    string
	Query         string
	Page          int
	PageSize      int
	LearnerUserID uint
}

type ImportKnowledgeBaseInput struct {
	Path               string
	SubjectKey         string
	Title              string
	OwnerLearnerUserID uint
}

type ImportKnowledgeBaseResult struct {
	Document       KnowledgeBaseDocument `json:"document"`
	ChunkCount     int                   `json:"chunk_count"`
	CharacterCount int                   `json:"character_count"`
}

type UpdateKnowledgeBaseDocumentStatusInput struct {
	Status string `json:"status"`
}

type PagedKnowledgeBaseDocuments struct {
	Items    []KnowledgeBaseDocument `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type PagedKnowledgeBaseChunks struct {
	Items    []KnowledgeBaseChunk `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}
