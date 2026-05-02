package domain

import "time"

type Subject struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Featured    bool   `json:"featured"`
}

type Category struct {
	ID          uint   `json:"id"`
	SubjectID   uint   `json:"subject_id"`
	SubjectKey  string `json:"subject_key,omitempty"`
	Kind        string `json:"kind"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Enabled     bool   `json:"enabled"`
}

type Grade struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Stage       string `json:"stage"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Enabled     bool   `json:"enabled"`
}

type Word struct {
	ID             uint64 `json:"id"`
	LegacyID       int64  `json:"legacy_id,omitempty"`
	SubjectID      uint   `json:"subject_id"`
	SubjectKey     string `json:"subject_key"`
	CategoryID     *uint  `json:"category_id,omitempty"`
	CategoryName   string `json:"category_name,omitempty"`
	GradeID        *uint  `json:"grade_id,omitempty"`
	GradeName      string `json:"grade_name,omitempty"`
	Term           string `json:"term"`
	Translation    string `json:"translation"`
	Classification string `json:"classification"`
	Source         string `json:"source,omitempty"`
	Phonetics      string `json:"phonetics,omitempty"`
	Explanation    string `json:"explanation,omitempty"`
	IsVIP          bool   `json:"is_vip"`
}

type ClassificationStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ClassificationStatFilter struct {
	SubjectKey string
	Page       int
	PageSize   int
}

type Plan struct {
	ID              uint     `json:"id"`
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	BillingMode     string   `json:"billing_mode"`
	PriceCents      int      `json:"price_cents"`
	Description     string   `json:"description"`
	Recommended     bool     `json:"recommended"`
	PaymentChannels []string `json:"payment_channels"`
	Features        []string `json:"features"`
}

type AdminUser struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	IsSuper     bool       `json:"is_super"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type AdminRole struct {
	ID          uint     `json:"id"`
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	System      bool     `json:"system"`
	Sort        int      `json:"sort"`
}

type LearnerUser struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminLoginInput struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

type AdminSession struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	Admin       AdminUser `json:"admin"`
}

type AdminRefreshInput struct {
	AccessToken string `json:"access_token"`
}

type LearnerLoginInput struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

type LearnerRegisterInput struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	DisplayName   string `json:"display_name"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

type LearnerSession struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresAt   time.Time   `json:"expires_at"`
	User        LearnerUser `json:"user"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type CreateAdminUserInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	IsSuper     *bool  `json:"is_super"`
}

type UpdateAdminUserInput struct {
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	IsSuper     *bool  `json:"is_super"`
}

type CreateAdminRoleInput struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Sort        int      `json:"sort"`
}

type UpdateAdminRoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Sort        int      `json:"sort"`
}

type AdminUserFilter struct {
	Query    string
	Role     string
	Status   string
	Page     int
	PageSize int
}

type CategoryFilter struct {
	SubjectKey string
	Kind       string
	Query      string
	Page       int
	PageSize   int
}

type GradeFilter struct {
	Stage    string
	Query    string
	Page     int
	PageSize int
}

type WordFilter struct {
	SubjectID      uint
	SubjectKey     string
	CategoryID     uint
	Classification string
	GradeID        uint
	Query          string
	Page           int
	PageSize       int
}

type PagedWords struct {
	Items    []Word `json:"items"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type PagedClassificationStats struct {
	Items    []ClassificationStat `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type PagedCategories struct {
	Items    []Category `json:"items"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

type PagedGrades struct {
	Items    []Grade `json:"items"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

type PagedAdminUsers struct {
	Items    []AdminUser `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type CatalogStats struct {
	SubjectCount          int64  `json:"subject_count"`
	WordCount             int64  `json:"word_count"`
	ClassificationCount   int64  `json:"classification_count"`
	GradeCount            int64  `json:"grade_count"`
	AdminCount            int64  `json:"admin_count"`
	DataSource            string `json:"data_source"`
	SampleData            bool   `json:"sample_data"`
	SuperAdminInitialized bool   `json:"super_admin_initialized"`
}

type CreateSubjectInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Featured    bool   `json:"featured"`
}

type UpdateSubjectInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Featured    bool   `json:"featured"`
}

type CreateCategoryInput struct {
	SubjectID   uint   `json:"subject_id"`
	SubjectKey  string `json:"subject_key"`
	Kind        string `json:"kind"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Enabled     *bool  `json:"enabled"`
}

type UpdateCategoryInput struct {
	SubjectID   uint   `json:"subject_id"`
	SubjectKey  string `json:"subject_key"`
	Kind        string `json:"kind"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Enabled     *bool  `json:"enabled"`
}

type CreateGradeInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Stage       string `json:"stage"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Enabled     *bool  `json:"enabled"`
}

type UpdateGradeInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Stage       string `json:"stage"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Enabled     *bool  `json:"enabled"`
}

type CreateWordInput struct {
	LegacyID       int64  `json:"legacy_id"`
	SubjectID      uint   `json:"subject_id"`
	SubjectKey     string `json:"subject_key"`
	CategoryID     *uint  `json:"category_id"`
	CategoryName   string `json:"category_name"`
	Classification string `json:"classification"`
	GradeID        *uint  `json:"grade_id"`
	Term           string `json:"term"`
	Translation    string `json:"translation"`
	Source         string `json:"source"`
	Phonetics      string `json:"phonetics"`
	Explanation    string `json:"explanation"`
	IsVIP          bool   `json:"is_vip"`
}

type UpdateWordInput struct {
	LegacyID       int64  `json:"legacy_id"`
	SubjectID      uint   `json:"subject_id"`
	SubjectKey     string `json:"subject_key"`
	CategoryID     *uint  `json:"category_id"`
	CategoryName   string `json:"category_name"`
	Classification string `json:"classification"`
	GradeID        *uint  `json:"grade_id"`
	Term           string `json:"term"`
	Translation    string `json:"translation"`
	Source         string `json:"source"`
	Phonetics      string `json:"phonetics"`
	Explanation    string `json:"explanation"`
	IsVIP          bool   `json:"is_vip"`
}

type CreatePlanInput struct {
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	BillingMode     string   `json:"billing_mode"`
	PriceCents      int      `json:"price_cents"`
	Description     string   `json:"description"`
	Recommended     bool     `json:"recommended"`
	PaymentChannels []string `json:"payment_channels"`
	Features        []string `json:"features"`
}

type UpdatePlanInput struct {
	Name            string   `json:"name"`
	BillingMode     string   `json:"billing_mode"`
	PriceCents      int      `json:"price_cents"`
	Description     string   `json:"description"`
	Recommended     bool     `json:"recommended"`
	PaymentChannels []string `json:"payment_channels"`
	Features        []string `json:"features"`
}

type ImportWordsInput struct {
	Path       string `json:"path"`
	SubjectKey string `json:"subject_key"`
	Replace    *bool  `json:"replace"`
}

type ImportResult struct {
	ImportedCount     int    `json:"imported_count"`
	CreatedCategories int    `json:"created_categories"`
	SubjectKey        string `json:"subject_key"`
	Path              string `json:"path"`
	Replace           bool   `json:"replace"`
}

type BootstrapAdminInput struct {
	Username    string
	Password    string
	DisplayName string
}
