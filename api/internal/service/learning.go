package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"brights/api/internal/domain"
	"brights/api/internal/storage"
)

const (
	learningLevelBeginner     = "beginner"
	learningLevelIntermediate = "intermediate"
	learningLevelAdvanced     = "advanced"
	learningLevelMastered     = "mastered"

	learningDifficultyEasy   = "easy"
	learningDifficultyMedium = "medium"
	learningDifficultyHard   = "hard"
)

type learningWordSnapshot struct {
	ID             uint64 `gorm:"column:id"`
	SubjectKey     string `gorm:"column:subject_key"`
	Term           string `gorm:"column:term"`
	Translation    string `gorm:"column:translation"`
	Classification string `gorm:"column:classification"`
	SourceLabel    string `gorm:"column:source_label"`
	Phonetics      string `gorm:"column:phonetics"`
	Explanation    string `gorm:"column:explanation"`
}

func (s *Service) ListLearnerWordProgress(
	ctx context.Context,
	learnerID uint,
	filter domain.LearnerWordProgressFilter,
) (domain.PagedLearnerWordProgress, error) {
	if learnerID == 0 {
		return domain.PagedLearnerWordProgress{}, errors.New("learner id is required")
	}

	page, pageSize := normalizePage(filter.Page, filter.PageSize, 20)
	query := s.db.WithContext(ctx).
		Model(&storage.LearnerWordProgress{}).
		Where("learner_user_id = ?", learnerID)

	if subjectKey := normalizeKey(filter.SubjectKey); subjectKey != "" {
		query = query.Where("subject_key = ?", subjectKey)
	}
	if level := strings.TrimSpace(strings.ToLower(filter.Level)); level != "" {
		query = query.Where("level = ?", level)
	}
	if difficulty := strings.TrimSpace(strings.ToLower(filter.Difficulty)); difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}
	if filter.DueOnly {
		query = query.Where("next_review_at IS NOT NULL AND next_review_at <= ?", time.Now())
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.
			Joins("JOIN words ON words.id = learner_word_progresses.word_id").
			Where(
				"words.term LIKE ? OR words.translation LIKE ? OR words.source_label LIKE ? OR words.explanation LIKE ?",
				like,
				like,
				like,
				like,
			)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return domain.PagedLearnerWordProgress{}, err
	}

	var models []storage.LearnerWordProgress
	if err := query.
		Order("CASE WHEN next_review_at IS NULL THEN 1 ELSE 0 END, next_review_at asc, updated_at desc, id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&models).Error; err != nil {
		return domain.PagedLearnerWordProgress{}, err
	}

	snapshots, err := s.learningWordSnapshotMap(ctx, models)
	if err != nil {
		return domain.PagedLearnerWordProgress{}, err
	}

	items := make([]domain.LearnerWordProgress, 0, len(models))
	for _, model := range models {
		items = append(items, toLearnerWordProgress(model, snapshots[model.WordID]))
	}

	return domain.PagedLearnerWordProgress{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) SaveLearnerWordProgress(
	ctx context.Context,
	learnerID uint,
	input domain.SaveLearnerWordProgressInput,
) (domain.LearnerWordProgress, error) {
	if learnerID == 0 {
		return domain.LearnerWordProgress{}, errors.New("learner id is required")
	}
	if input.WordID == 0 {
		return domain.LearnerWordProgress{}, errors.New("word id is required")
	}

	snapshot, err := s.findLearningWordSnapshot(ctx, input.WordID)
	if err != nil {
		return domain.LearnerWordProgress{}, err
	}
	if subjectKey := normalizeKey(input.SubjectKey); subjectKey != "" && subjectKey != snapshot.SubjectKey {
		return domain.LearnerWordProgress{}, errors.New("word does not belong to the selected subject")
	}

	var model storage.LearnerWordProgress
	err = s.db.WithContext(ctx).
		Where("learner_user_id = ? AND word_id = ?", learnerID, input.WordID).
		First(&model).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		model = storage.LearnerWordProgress{
			LearnerUserID: learnerID,
			WordID:        input.WordID,
			SubjectKey:    snapshot.SubjectKey,
			Level:         learningLevelBeginner,
			Difficulty:    learningDifficultyMedium,
		}
	case err != nil:
		return domain.LearnerWordProgress{}, err
	}

	if level := strings.TrimSpace(input.Level); level != "" {
		model.Level, err = normalizeLearningLevel(level)
		if err != nil {
			return domain.LearnerWordProgress{}, err
		}
	}
	if difficulty := strings.TrimSpace(input.Difficulty); difficulty != "" {
		model.Difficulty, err = normalizeLearningDifficulty(difficulty)
		if err != nil {
			return domain.LearnerWordProgress{}, err
		}
	}
	model.SubjectKey = snapshot.SubjectKey

	now := time.Now()
	if model.Level == learningLevelMastered {
		model.MasteredAt = &now
	} else {
		model.MasteredAt = nil
	}

	if err := s.db.WithContext(ctx).Save(&model).Error; err != nil {
		return domain.LearnerWordProgress{}, err
	}
	return toLearnerWordProgress(model, snapshot), nil
}

func (s *Service) ReviewLearnerWord(
	ctx context.Context,
	learnerID uint,
	input domain.ReviewLearnerWordInput,
) (domain.LearnerWordProgress, error) {
	if learnerID == 0 {
		return domain.LearnerWordProgress{}, errors.New("learner id is required")
	}
	if input.WordID == 0 {
		return domain.LearnerWordProgress{}, errors.New("word id is required")
	}

	snapshot, err := s.findLearningWordSnapshot(ctx, input.WordID)
	if err != nil {
		return domain.LearnerWordProgress{}, err
	}
	if subjectKey := normalizeKey(input.SubjectKey); subjectKey != "" && subjectKey != snapshot.SubjectKey {
		return domain.LearnerWordProgress{}, errors.New("word does not belong to the selected subject")
	}

	var saved storage.LearnerWordProgress
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model storage.LearnerWordProgress
		err := tx.Where("learner_user_id = ? AND word_id = ?", learnerID, input.WordID).First(&model).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			model = storage.LearnerWordProgress{
				LearnerUserID: learnerID,
				WordID:        input.WordID,
				SubjectKey:    snapshot.SubjectKey,
				Level:         learningLevelBeginner,
				Difficulty:    learningDifficultyMedium,
			}
		case err != nil:
			return err
		}

		if model.SubjectKey == "" {
			model.SubjectKey = snapshot.SubjectKey
		}

		level, err := normalizeLearningLevel(firstNonEmpty(input.Level, model.Level, learningLevelBeginner))
		if err != nil {
			return err
		}
		difficulty, err := normalizeLearningDifficulty(firstNonEmpty(input.Difficulty, model.Difficulty, learningDifficultyMedium))
		if err != nil {
			return err
		}

		now := time.Now()
		model.SubjectKey = snapshot.SubjectKey
		model.ReviewCount++
		model.LastReviewedAt = &now
		model.Difficulty = difficulty

		if input.Remembered {
			model.CorrectCount++
			model.ConsecutiveCorrect++
		} else {
			model.IncorrectCount++
			model.ConsecutiveCorrect = 0
		}

		if strings.TrimSpace(input.Level) == "" {
			level = learningNextLevel(level, input.Remembered, model.ConsecutiveCorrect)
		}
		model.Level = level

		nextReview := now.Add(learningNextReviewInterval(level, difficulty, input.Remembered))
		model.NextReviewAt = &nextReview

		if model.Level == learningLevelMastered {
			model.MasteredAt = &now
		} else {
			model.MasteredAt = nil
		}

		if err := tx.Save(&model).Error; err != nil {
			return err
		}

		logModel := storage.LearnerWordReviewLog{
			LearnerUserID: learnerID,
			WordID:        input.WordID,
			SubjectKey:    snapshot.SubjectKey,
			Level:         model.Level,
			Difficulty:    model.Difficulty,
			Result:        learningReviewResultLabel(input.Remembered),
			ReviewedAt:    now,
			NextReviewAt:  model.NextReviewAt,
		}
		if err := tx.Create(&logModel).Error; err != nil {
			return err
		}

		saved = model
		return nil
	}); err != nil {
		return domain.LearnerWordProgress{}, err
	}

	return toLearnerWordProgress(saved, snapshot), nil
}

func (s *Service) GetLearnerLearningSummary(
	ctx context.Context,
	learnerID uint,
	subjectKey string,
) (domain.LearningSummary, error) {
	if learnerID == 0 {
		return domain.LearningSummary{}, errors.New("learner id is required")
	}

	subjectKey = normalizeKey(subjectKey)
	baseQuery := s.db.WithContext(ctx).
		Model(&storage.LearnerWordProgress{}).
		Where("learner_user_id = ?", learnerID)
	if subjectKey != "" {
		baseQuery = baseQuery.Where("subject_key = ?", subjectKey)
	}

	var trackedWords int64
	if err := baseQuery.Count(&trackedWords).Error; err != nil {
		return domain.LearningSummary{}, err
	}

	now := time.Now()
	var dueReviews int64
	if err := baseQuery.Session(&gorm.Session{}).
		Where("next_review_at IS NOT NULL AND next_review_at <= ?", now).
		Count(&dueReviews).Error; err != nil {
		return domain.LearningSummary{}, err
	}

	var stats struct {
		ReviewCount   int64
		CorrectCount  int64
		MasteredWords int64
	}
	if err := baseQuery.Session(&gorm.Session{}).
		Select(
			"COALESCE(SUM(review_count), 0) AS review_count, " +
				"COALESCE(SUM(correct_count), 0) AS correct_count, " +
				"COALESCE(SUM(CASE WHEN level = 'mastered' THEN 1 ELSE 0 END), 0) AS mastered_words",
		).
		Scan(&stats).Error; err != nil {
		return domain.LearningSummary{}, err
	}

	levelCounts, err := s.learningCountItems(ctx, learnerID, subjectKey, "level", []domain.LearningCountItem{
		{Key: learningLevelBeginner, Label: "初级"},
		{Key: learningLevelIntermediate, Label: "中级"},
		{Key: learningLevelAdvanced, Label: "高级"},
		{Key: learningLevelMastered, Label: "已掌握"},
	})
	if err != nil {
		return domain.LearningSummary{}, err
	}

	difficultyCounts, err := s.learningCountItems(ctx, learnerID, subjectKey, "difficulty", []domain.LearningCountItem{
		{Key: learningDifficultyEasy, Label: "简单"},
		{Key: learningDifficultyMedium, Label: "中等"},
		{Key: learningDifficultyHard, Label: "困难"},
	})
	if err != nil {
		return domain.LearningSummary{}, err
	}

	curvePoints, err := s.learningCurvePoints(ctx, learnerID, subjectKey, 14)
	if err != nil {
		return domain.LearningSummary{}, err
	}

	correctRate := 0.0
	if stats.ReviewCount > 0 {
		correctRate = float64(stats.CorrectCount) / float64(stats.ReviewCount)
	}

	return domain.LearningSummary{
		SubjectKey:       firstNonEmpty(subjectKey, "english"),
		TrackedWords:     trackedWords,
		DueReviews:       dueReviews,
		MasteredWords:    stats.MasteredWords,
		ReviewCount:      stats.ReviewCount,
		CorrectRate:      correctRate,
		LevelCounts:      levelCounts,
		DifficultyCounts: difficultyCounts,
		CurvePoints:      curvePoints,
	}, nil
}

func (s *Service) learningCountItems(
	ctx context.Context,
	learnerID uint,
	subjectKey string,
	field string,
	defaults []domain.LearningCountItem,
) ([]domain.LearningCountItem, error) {
	type row struct {
		BucketKey string `gorm:"column:bucket_key"`
		Count     int64  `gorm:"column:total_count"`
	}

	query := s.db.WithContext(ctx).
		Model(&storage.LearnerWordProgress{}).
		Select(field+" AS bucket_key, COUNT(*) AS total_count").
		Where("learner_user_id = ?", learnerID)
	if subjectKey != "" {
		query = query.Where("subject_key = ?", subjectKey)
	}

	var rows []row
	if err := query.Group(field).Scan(&rows).Error; err != nil {
		return nil, err
	}

	countMap := make(map[string]int64, len(rows))
	for _, item := range rows {
		countMap[strings.TrimSpace(strings.ToLower(item.BucketKey))] = item.Count
	}

	items := make([]domain.LearningCountItem, 0, len(defaults))
	for _, item := range defaults {
		item.Count = countMap[item.Key]
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) learningCurvePoints(
	ctx context.Context,
	learnerID uint,
	subjectKey string,
	days int,
) ([]domain.LearningCurvePoint, error) {
	if days <= 0 {
		days = 14
	}

	start := time.Now().AddDate(0, 0, -(days - 1))
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())

	var logs []storage.LearnerWordReviewLog
	query := s.db.WithContext(ctx).
		Where("learner_user_id = ? AND reviewed_at >= ?", learnerID, start)
	if subjectKey != "" {
		query = query.Where("subject_key = ?", subjectKey)
	}
	if err := query.Order("reviewed_at asc").Find(&logs).Error; err != nil {
		return nil, err
	}

	type counters struct {
		reviews   int64
		correct   int64
		incorrect int64
	}
	countMap := make(map[string]counters, days)
	for _, item := range logs {
		key := item.ReviewedAt.Format("2006-01-02")
		current := countMap[key]
		current.reviews++
		if item.Result == learningReviewResultLabel(true) {
			current.correct++
		} else {
			current.incorrect++
		}
		countMap[key] = current
	}

	points := make([]domain.LearningCurvePoint, 0, days)
	for dayOffset := 0; dayOffset < days; dayOffset++ {
		day := start.AddDate(0, 0, dayOffset)
		key := day.Format("2006-01-02")
		value := countMap[key]
		retentionRate := 0.0
		if value.reviews > 0 {
			retentionRate = float64(value.correct) / float64(value.reviews)
		}
		points = append(points, domain.LearningCurvePoint{
			Date:           key,
			ReviewCount:    value.reviews,
			CorrectCount:   value.correct,
			IncorrectCount: value.incorrect,
			RetentionRate:  retentionRate,
		})
	}
	return points, nil
}

func (s *Service) findLearningWordSnapshot(ctx context.Context, wordID uint64) (learningWordSnapshot, error) {
	var snapshot learningWordSnapshot
	err := s.db.WithContext(ctx).
		Table("words").
		Select(
			"words.id, subjects.subject_key, words.term, words.translation, "+
				"words.source_label AS source_label, words.phonetics, words.explanation, "+
				"COALESCE(categories.name, 'Unclassified') AS classification",
		).
		Joins("JOIN subjects ON subjects.id = words.subject_id").
		Joins("LEFT JOIN categories ON categories.id = words.category_id").
		Where("words.id = ?", wordID).
		Scan(&snapshot).Error
	if err != nil {
		return learningWordSnapshot{}, err
	}
	if snapshot.ID == 0 {
		return learningWordSnapshot{}, errors.New("word does not exist")
	}
	snapshot.SubjectKey = normalizeKey(snapshot.SubjectKey)
	return snapshot, nil
}

func (s *Service) learningWordSnapshotMap(
	ctx context.Context,
	models []storage.LearnerWordProgress,
) (map[uint64]learningWordSnapshot, error) {
	result := make(map[uint64]learningWordSnapshot)
	if len(models) == 0 {
		return result, nil
	}

	ids := make([]uint64, 0, len(models))
	seen := make(map[uint64]struct{}, len(models))
	for _, model := range models {
		if model.WordID == 0 {
			continue
		}
		if _, ok := seen[model.WordID]; ok {
			continue
		}
		seen[model.WordID] = struct{}{}
		ids = append(ids, model.WordID)
	}

	var items []learningWordSnapshot
	if err := s.db.WithContext(ctx).
		Table("words").
		Select(
			"words.id, subjects.subject_key, words.term, words.translation, "+
				"words.source_label AS source_label, words.phonetics, words.explanation, "+
				"COALESCE(categories.name, 'Unclassified') AS classification",
		).
		Joins("JOIN subjects ON subjects.id = words.subject_id").
		Joins("LEFT JOIN categories ON categories.id = words.category_id").
		Where("words.id IN ?", ids).
		Scan(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		item.SubjectKey = normalizeKey(item.SubjectKey)
		result[item.ID] = item
	}
	return result, nil
}

func toLearnerWordProgress(model storage.LearnerWordProgress, snapshot learningWordSnapshot) domain.LearnerWordProgress {
	isDue := model.NextReviewAt != nil && !model.NextReviewAt.After(time.Now())
	return domain.LearnerWordProgress{
		ID:                 model.ID,
		LearnerUserID:      model.LearnerUserID,
		WordID:             model.WordID,
		SubjectKey:         firstNonEmpty(snapshot.SubjectKey, model.SubjectKey),
		Term:               snapshot.Term,
		Translation:        snapshot.Translation,
		Classification:     snapshot.Classification,
		Source:             snapshot.SourceLabel,
		Phonetics:          snapshot.Phonetics,
		Explanation:        snapshot.Explanation,
		Level:              model.Level,
		Difficulty:         model.Difficulty,
		ReviewCount:        model.ReviewCount,
		CorrectCount:       model.CorrectCount,
		IncorrectCount:     model.IncorrectCount,
		ConsecutiveCorrect: model.ConsecutiveCorrect,
		LastReviewedAt:     model.LastReviewedAt,
		NextReviewAt:       model.NextReviewAt,
		MasteredAt:         model.MasteredAt,
		IsDue:              isDue,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func normalizeLearningLevel(value string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", learningLevelBeginner:
		return learningLevelBeginner, nil
	case learningLevelIntermediate:
		return learningLevelIntermediate, nil
	case learningLevelAdvanced:
		return learningLevelAdvanced, nil
	case learningLevelMastered:
		return learningLevelMastered, nil
	default:
		return "", errors.New("learning level must be beginner, intermediate, advanced, or mastered")
	}
}

func normalizeLearningDifficulty(value string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", learningDifficultyMedium:
		return learningDifficultyMedium, nil
	case learningDifficultyEasy:
		return learningDifficultyEasy, nil
	case learningDifficultyHard:
		return learningDifficultyHard, nil
	default:
		return "", errors.New("learning difficulty must be easy, medium, or hard")
	}
}

func learningNextLevel(current string, remembered bool, consecutiveCorrect int) string {
	current, err := normalizeLearningLevel(current)
	if err != nil {
		current = learningLevelBeginner
	}

	if !remembered {
		switch current {
		case learningLevelMastered:
			return learningLevelAdvanced
		case learningLevelAdvanced:
			return learningLevelIntermediate
		case learningLevelIntermediate:
			return learningLevelBeginner
		default:
			return learningLevelBeginner
		}
	}

	switch current {
	case learningLevelBeginner:
		if consecutiveCorrect >= 2 {
			return learningLevelIntermediate
		}
	case learningLevelIntermediate:
		if consecutiveCorrect >= 4 {
			return learningLevelAdvanced
		}
	case learningLevelAdvanced:
		if consecutiveCorrect >= 6 {
			return learningLevelMastered
		}
	}
	return current
}

func learningNextReviewInterval(level string, difficulty string, remembered bool) time.Duration {
	base := 24 * time.Hour

	switch level {
	case learningLevelIntermediate:
		if remembered {
			base = 72 * time.Hour
		} else {
			base = 12 * time.Hour
		}
	case learningLevelAdvanced:
		if remembered {
			base = 7 * 24 * time.Hour
		} else {
			base = 24 * time.Hour
		}
	case learningLevelMastered:
		if remembered {
			base = 21 * 24 * time.Hour
		} else {
			base = 48 * time.Hour
		}
	default:
		if remembered {
			base = 24 * time.Hour
		} else {
			base = 6 * time.Hour
		}
	}

	switch difficulty {
	case learningDifficultyEasy:
		return time.Duration(float64(base) * 1.5)
	case learningDifficultyHard:
		return time.Duration(float64(base) * 0.6)
	default:
		return base
	}
}

func learningReviewResultLabel(remembered bool) string {
	if remembered {
		return "remembered"
	}
	return "forgot"
}
