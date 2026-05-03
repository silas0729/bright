package storage

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	gosqlmysql "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type JSONStringSlice []string

func (s JSONStringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (s *JSONStringSlice) Scan(value any) error {
	if value == nil {
		*s = JSONStringSlice{}
		return nil
	}

	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return errors.New("unsupported JSONStringSlice type")
	}

	if len(raw) == 0 {
		*s = JSONStringSlice{}
		return nil
	}

	var items []string
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	*s = JSONStringSlice(items)
	return nil
}

type Subject struct {
	ID          uint   `gorm:"primaryKey"`
	Key         string `gorm:"column:subject_key;size:80;uniqueIndex;not null"`
	Name        string `gorm:"size:120;not null"`
	Description string `gorm:"size:500"`
	Sort        int    `gorm:"not null;default:0"`
	Featured    bool   `gorm:"not null;default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Category struct {
	ID          uint   `gorm:"primaryKey"`
	SubjectID   uint   `gorm:"not null;index;uniqueIndex:idx_subject_kind_category_key,priority:1"`
	Kind        string `gorm:"size:32;not null;default:topic;uniqueIndex:idx_subject_kind_category_key,priority:2"`
	Key         string `gorm:"column:category_key;size:120;not null;uniqueIndex:idx_subject_kind_category_key,priority:3"`
	Name        string `gorm:"size:120;not null;index"`
	Description string `gorm:"size:500"`
	Sort        int    `gorm:"not null;default:0"`
	Enabled     bool   `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Grade struct {
	ID          uint   `gorm:"primaryKey"`
	Key         string `gorm:"column:grade_key;size:80;uniqueIndex;not null"`
	Name        string `gorm:"size:120;not null"`
	Stage       string `gorm:"size:80;index"`
	Description string `gorm:"size:500"`
	Sort        int    `gorm:"not null;default:0"`
	Enabled     bool   `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Word struct {
	ID          uint64 `gorm:"primaryKey"`
	LegacyID    int64  `gorm:"index"`
	SubjectID   uint   `gorm:"not null;index;index:idx_words_subject_category,priority:1"`
	CategoryID  *uint  `gorm:"index;index:idx_words_subject_category,priority:2"`
	GradeID     *uint  `gorm:"index"`
	Term        string `gorm:"size:180;not null;index"`
	Translation string `gorm:"size:255;index"`
	SourceLabel string `gorm:"size:255;index"`
	Phonetics   string `gorm:"type:text"`
	Explanation string `gorm:"type:text"`
	IsVIP       bool   `gorm:"column:is_v_ip;not null;default:false;index"`
	Status      string `gorm:"size:32;not null;default:published;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Subject  Subject   `gorm:"foreignKey:SubjectID"`
	Category *Category `gorm:"foreignKey:CategoryID"`
	Grade    *Grade    `gorm:"foreignKey:GradeID"`
}

type ClassificationSummary struct {
	ID            uint   `gorm:"primaryKey"`
	SubjectID     uint   `gorm:"not null;index;uniqueIndex:idx_subject_classification_name,priority:1"`
	Name          string `gorm:"size:120;not null;uniqueIndex:idx_subject_classification_name,priority:2"`
	WordCount     int64  `gorm:"column:word_count;not null;default:0;index"`
	FreeWordCount int64  `gorm:"column:free_word_count;not null;default:0;index"`
	VIPWordCount  int64  `gorm:"column:vip_word_count;not null;default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Plan struct {
	ID              uint            `gorm:"primaryKey"`
	Key             string          `gorm:"column:plan_key;size:80;uniqueIndex;not null"`
	Name            string          `gorm:"size:120;not null"`
	BillingMode     string          `gorm:"size:40;not null;index"`
	PriceCents      int             `gorm:"not null"`
	Description     string          `gorm:"size:500"`
	Recommended     bool            `gorm:"not null;default:false"`
	PaymentChannels JSONStringSlice `gorm:"type:text;not null"`
	Features        JSONStringSlice `gorm:"type:text;not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type AdminUser struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"size:80;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	DisplayName  string `gorm:"size:120;not null"`
	Role         string `gorm:"size:40;not null;default:super_admin"`
	IsSuper      bool   `gorm:"not null;default:false"`
	Status       string `gorm:"size:32;not null;default:active"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AdminRole struct {
	ID          uint            `gorm:"primaryKey"`
	Key         string          `gorm:"column:role_key;size:80;uniqueIndex;not null"`
	Name        string          `gorm:"size:120;not null"`
	Description string          `gorm:"size:500"`
	Permissions JSONStringSlice `gorm:"type:text;not null"`
	System      bool            `gorm:"not null;default:false"`
	Sort        int             `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type LearnerUser struct {
	ID              uint   `gorm:"primaryKey"`
	Username        string `gorm:"size:80;uniqueIndex;not null"`
	PasswordHash    string `gorm:"size:255;not null"`
	DisplayName     string `gorm:"size:120;not null"`
	Status          string `gorm:"size:32;not null;default:active"`
	InviteCode      string `gorm:"size:80;uniqueIndex;not null;default:''"`
	InvitedByUserID *uint  `gorm:"index"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SiteSetting struct {
	ID              uint   `gorm:"primaryKey"`
	SiteName        string `gorm:"size:120;not null"`
	SiteIcon        string `gorm:"type:longtext"`
	SiteTagline     string `gorm:"size:255"`
	HeroTitle       string `gorm:"size:255;not null"`
	HeroDescription string `gorm:"type:text"`
	SEOHeadline     string `gorm:"size:255"`
	SEOTitle        string `gorm:"size:255;not null"`
	SEODescription  string `gorm:"type:text"`
	SEOKeywords     string `gorm:"type:text"`
	FooterText      string `gorm:"type:text"`
	ContactEmail    string `gorm:"size:120"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type WechatPayConfig struct {
	ID                     uint   `gorm:"primaryKey"`
	AuthMode               string `gorm:"size:32;not null;default:public_key"`
	MchID                  string `gorm:"size:64;not null"`
	AppID                  string `gorm:"size:128"`
	MerchantSerialNo       string `gorm:"size:128;not null"`
	APIv3KeyEnc            string `gorm:"type:text"`
	PlatformCertSerialNo   string `gorm:"size:128"`
	NotifyURL              string `gorm:"size:500"`
	DescriptionPrefix      string `gorm:"size:120"`
	TimeExpireMinutes      int    `gorm:"not null;default:30"`
	WechatPayPublicKeyID   string `gorm:"size:128"`
	WechatPayPublicKeyPath string `gorm:"type:text"`
	P12Path                string `gorm:"type:text"`
	CertPemPath            string `gorm:"type:text"`
	KeyPemPath             string `gorm:"type:text"`
	PlatformCertPath       string `gorm:"type:text"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type PaymentOrder struct {
	ID              uint   `gorm:"primaryKey"`
	OrderNo         string `gorm:"size:80;uniqueIndex;not null"`
	PlanID          *uint  `gorm:"index"`
	PlanKey         string `gorm:"size:80;index"`
	SubjectKey      string `gorm:"size:80;index"`
	CustomerRef     string `gorm:"size:120;not null;index"`
	Description     string `gorm:"size:255"`
	BillingMode     string `gorm:"size:40;not null;index"`
	AmountCents     int    `gorm:"not null"`
	Currency        string `gorm:"size:12;not null;default:CNY"`
	Provider        string `gorm:"size:40;not null;default:wechat"`
	ProviderTradeNo string `gorm:"size:120;index"`
	CodeURL         string `gorm:"size:500"`
	Status          string `gorm:"size:32;not null;default:pending;index"`
	ErrorMessage    string `gorm:"type:text"`
	PaidAt          *time.Time
	ExpiresAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MemberSubscription struct {
	ID                 uint   `gorm:"primaryKey"`
	CustomerRef        string `gorm:"size:120;not null;index"`
	PlanID             *uint  `gorm:"index"`
	PlanKey            string `gorm:"size:80;index"`
	SubjectKey         string `gorm:"size:80;index"`
	Status             string `gorm:"size:32;not null;default:pending;index"`
	AutoRenew          bool   `gorm:"not null;default:false"`
	Provider           string `gorm:"size:40;not null;default:wechat"`
	ProviderContractID string `gorm:"size:120;index"`
	StartedAt          *time.Time
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ImportJob struct {
	ID            uint   `gorm:"primaryKey"`
	SubjectID     uint   `gorm:"not null;index"`
	SourcePath    string `gorm:"size:500;not null"`
	SourceName    string `gorm:"size:255;not null"`
	ImportedCount int    `gorm:"not null;default:0"`
	ReplaceMode   bool   `gorm:"not null;default:true"`
	Status        string `gorm:"size:32;not null;default:running"`
	ErrorMessage  string `gorm:"type:text"`
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type KnowledgeBaseDocument struct {
	ID             uint   `gorm:"primaryKey"`
	SubjectKey     string `gorm:"size:80;not null;index"`
	Title          string `gorm:"size:255;not null;index"`
	SourceFileName string `gorm:"size:255;not null"`
	SourceType     string `gorm:"size:32;not null;index"`
	Status         string `gorm:"size:32;not null;default:active;index"`
	ChunkCount     int    `gorm:"not null;default:0"`
	CharacterCount int    `gorm:"not null;default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type KnowledgeBaseChunk struct {
	ID             uint   `gorm:"primaryKey"`
	DocumentID     uint   `gorm:"not null;index;index:idx_kb_document_chunk,priority:1"`
	SubjectKey     string `gorm:"size:80;not null;index"`
	Title          string `gorm:"size:255;not null;index"`
	ChunkIndex     int    `gorm:"not null;index:idx_kb_document_chunk,priority:2"`
	Content        string `gorm:"type:text;not null"`
	ContentSearch  string `gorm:"type:text"`
	CharacterCount int    `gorm:"not null;default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MCPToolConfig struct {
	ID                 uint   `gorm:"primaryKey"`
	ToolName           string `gorm:"column:tool_name;size:120;uniqueIndex;not null"`
	Title              string `gorm:"size:160;not null"`
	Description        string `gorm:"size:500"`
	Category           string `gorm:"size:80;not null;default:general;index"`
	SourceType         string `gorm:"size:40;not null;default:builtin;index"`
	IsEnabled          bool   `gorm:"column:is_enabled;not null;default:true;index"`
	RequiresMembership bool   `gorm:"column:requires_membership;not null;default:false;index"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func Open(driverName, dsn string, autoCreateDatabase bool) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch strings.ToLower(strings.TrimSpace(driverName)) {
	case "", "sqlite":
		resolved, err := ensureSQLitePath(dsn)
		if err != nil {
			return nil, err
		}
		return gorm.Open(sqlite.Open(resolved), gormConfig)
	case "mysql":
		if autoCreateDatabase {
			if err := ensureMySQLDatabase(dsn); err != nil {
				return nil, err
			}
		}
		return gorm.Open(gormmysql.Open(dsn), gormConfig)
	default:
		return nil, errors.New("unsupported database driver")
	}
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Subject{},
		&Category{},
		&Grade{},
		&Word{},
		&ClassificationSummary{},
		&Plan{},
		&AdminUser{},
		&AdminRole{},
		&LearnerUser{},
		&SiteSetting{},
		&WechatPayConfig{},
		&PaymentOrder{},
		&MemberSubscription{},
		&LearnerMCPEndpoint{},
		&ImportJob{},
		&KnowledgeBaseDocument{},
		&KnowledgeBaseChunk{},
		&MCPToolConfig{},
	)
}

func ensureSQLitePath(dsn string) (string, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		dsn = filepath.Join(".", "brights.db")
	}
	if strings.HasPrefix(strings.ToLower(dsn), "file:") {
		return dsn, nil
	}
	dir := filepath.Dir(dsn)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return dsn, nil
}

func ensureMySQLDatabase(dsn string) error {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return errors.New("mysql dsn is required")
	}

	cfg, err := gosqlmysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.DBName) == "" {
		return errors.New("mysql database name is required")
	}

	dbName := cfg.DBName
	cfg.DBName = ""

	rawDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer rawDB.Close()

	if err := rawDB.Ping(); err != nil {
		return err
	}

	escaped := strings.ReplaceAll(dbName, "`", "``")
	statement := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		escaped,
	)
	_, err = rawDB.Exec(statement)
	return err
}

func MigrateLegacySchema(db *gorm.DB, driverName string) error {
	switch strings.ToLower(strings.TrimSpace(driverName)) {
	case "mysql":
		return migrateLegacyMySQLSchema(db)
	default:
		return nil
	}
}

type legacyColumnStep struct {
	Table         string
	OldColumn     string
	NewColumn     string
	NewDefinition string
}

func migrateLegacyMySQLSchema(db *gorm.DB) error {
	steps := []legacyColumnStep{
		{Table: "subjects", OldColumn: "key", NewColumn: "subject_key", NewDefinition: "VARCHAR(80) NOT NULL"},
		{Table: "categories", OldColumn: "key", NewColumn: "category_key", NewDefinition: "VARCHAR(120) NOT NULL"},
		{Table: "grades", OldColumn: "key", NewColumn: "grade_key", NewDefinition: "VARCHAR(80) NOT NULL"},
		{Table: "plans", OldColumn: "key", NewColumn: "plan_key", NewDefinition: "VARCHAR(80) NOT NULL"},
		{Table: "admin_roles", OldColumn: "key", NewColumn: "role_key", NewDefinition: "VARCHAR(80) NOT NULL"},
	}

	for _, step := range steps {
		if err := migrateLegacyColumn(db, step); err != nil {
			return err
		}
	}

	if err := migrateMySQLColumnDefinition(db, "words", "phonetics", "TEXT"); err != nil {
		return err
	}
	if err := migrateMySQLColumnDefinition(db, "admin_users", "is_super", "TINYINT(1) NOT NULL DEFAULT 0"); err != nil {
		return err
	}

	return nil
}

func migrateLegacyColumn(db *gorm.DB, step legacyColumnStep) error {
	oldExists, err := hasColumn(db, step.Table, step.OldColumn)
	if err != nil {
		return err
	}
	newExists, err := hasColumn(db, step.Table, step.NewColumn)
	if err != nil {
		return err
	}

	switch {
	case oldExists && !newExists:
		query := fmt.Sprintf(
			"ALTER TABLE `%s` CHANGE COLUMN `%s` `%s` %s",
			step.Table,
			step.OldColumn,
			step.NewColumn,
			step.NewDefinition,
		)
		return db.Exec(query).Error
	case oldExists && newExists:
		update := fmt.Sprintf(
			"UPDATE `%s` SET `%s` = `%s` WHERE (`%s` IS NULL OR `%s` = '') AND `%s` IS NOT NULL",
			step.Table,
			step.NewColumn,
			step.OldColumn,
			step.NewColumn,
			step.NewColumn,
			step.OldColumn,
		)
		if err := db.Exec(update).Error; err != nil {
			return err
		}
		drop := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", step.Table, step.OldColumn)
		return db.Exec(drop).Error
	default:
		return nil
	}
}

func hasColumn(db *gorm.DB, tableName, columnName string) (bool, error) {
	type result struct {
		Count int64
	}
	var row result
	if err := db.Raw(
		`SELECT COUNT(*) AS count
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
		tableName,
		columnName,
	).Scan(&row).Error; err != nil {
		return false, err
	}
	return row.Count > 0, nil
}

func migrateMySQLColumnDefinition(db *gorm.DB, tableName, columnName, definition string) error {
	exists, err := hasColumn(db, tableName, columnName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	query := fmt.Sprintf(
		"ALTER TABLE `%s` MODIFY COLUMN `%s` %s",
		tableName,
		columnName,
		definition,
	)
	return db.Exec(query).Error
}
