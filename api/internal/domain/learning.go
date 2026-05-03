package domain

import "time"

type LearnerWordProgress struct {
	ID                 uint       `json:"id"`
	LearnerUserID      uint       `json:"learner_user_id"`
	WordID             uint64     `json:"word_id"`
	SubjectKey         string     `json:"subject_key"`
	Term               string     `json:"term"`
	Translation        string     `json:"translation"`
	Classification     string     `json:"classification"`
	Source             string     `json:"source,omitempty"`
	Phonetics          string     `json:"phonetics,omitempty"`
	Explanation        string     `json:"explanation,omitempty"`
	Level              string     `json:"level"`
	Difficulty         string     `json:"difficulty"`
	ReviewCount        int        `json:"review_count"`
	CorrectCount       int        `json:"correct_count"`
	IncorrectCount     int        `json:"incorrect_count"`
	ConsecutiveCorrect int        `json:"consecutive_correct"`
	LastReviewedAt     *time.Time `json:"last_reviewed_at,omitempty"`
	NextReviewAt       *time.Time `json:"next_review_at,omitempty"`
	MasteredAt         *time.Time `json:"mastered_at,omitempty"`
	IsDue              bool       `json:"is_due"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type LearnerWordProgressFilter struct {
	SubjectKey    string
	Query         string
	Level         string
	Difficulty    string
	Page          int
	PageSize      int
	DueOnly       bool
	LearnerUserID uint
}

type SaveLearnerWordProgressInput struct {
	WordID     uint64 `json:"word_id"`
	SubjectKey string `json:"subject_key"`
	Level      string `json:"level"`
	Difficulty string `json:"difficulty"`
}

type ReviewLearnerWordInput struct {
	WordID     uint64 `json:"word_id"`
	SubjectKey string `json:"subject_key"`
	Remembered bool   `json:"remembered"`
	Level      string `json:"level"`
	Difficulty string `json:"difficulty"`
}

type LearningCountItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type LearningCurvePoint struct {
	Date           string  `json:"date"`
	ReviewCount    int64   `json:"review_count"`
	CorrectCount   int64   `json:"correct_count"`
	IncorrectCount int64   `json:"incorrect_count"`
	RetentionRate  float64 `json:"retention_rate"`
}

type LearningSummary struct {
	SubjectKey       string               `json:"subject_key"`
	TrackedWords     int64                `json:"tracked_words"`
	DueReviews       int64                `json:"due_reviews"`
	MasteredWords    int64                `json:"mastered_words"`
	ReviewCount      int64                `json:"review_count"`
	CorrectRate      float64              `json:"correct_rate"`
	LevelCounts      []LearningCountItem  `json:"level_counts"`
	DifficultyCounts []LearningCountItem  `json:"difficulty_counts"`
	CurvePoints      []LearningCurvePoint `json:"curve_points"`
}

type PagedLearnerWordProgress struct {
	Items    []LearnerWordProgress `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}
