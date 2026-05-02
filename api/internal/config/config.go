package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"brights/api/internal/catalog"
)

type Config struct {
	SourceFile string          `json:"-"`
	App        AppConfig       `json:"app"`
	Auth       AuthConfig      `json:"auth"`
	Database   DatabaseConfig  `json:"database"`
	Bootstrap  BootstrapConfig `json:"bootstrap"`
	Import     ImportConfig    `json:"import"`
}

type AppConfig struct {
	Addr string `json:"addr"`
}

type AuthConfig struct {
	Issuer                string `json:"issuer"`
	JWTSecret             string `json:"jwt_secret"`
	AccessTokenTTLMinutes int    `json:"access_token_ttl_minutes"`
}

type DatabaseConfig struct {
	Driver             string `json:"driver"`
	DSN                string `json:"dsn"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Name               string `json:"name"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	Charset            string `json:"charset"`
	ParseTime          bool   `json:"parse_time"`
	Loc                string `json:"loc"`
	AutoCreateDatabase bool   `json:"auto_create_database"`
}

type BootstrapConfig struct {
	SuperAdmin SuperAdminConfig `json:"super_admin"`
}

type SuperAdminConfig struct {
	Enabled     bool   `json:"enabled"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type ImportConfig struct {
	Enabled     bool   `json:"enabled"`
	SubjectKey  string `json:"subject_key"`
	DataFile    string `json:"data_file"`
	Replace     bool   `json:"replace"`
	OnlyIfEmpty bool   `json:"only_if_empty"`
}

func Load(configPath string) (Config, error) {
	cfg := defaultConfig()

	resolvedPath, err := resolveConfigPath(configPath)
	if err != nil {
		return Config{}, err
	}
	if resolvedPath != "" {
		if err := mergeJSONConfig(&cfg, resolvedPath); err != nil {
			return Config{}, err
		}
		cfg.SourceFile = resolvedPath
	}

	applyEnvOverrides(&cfg)
	cfg.normalize()

	if cfg.Import.DataFile == "" {
		if detected, ok := catalog.FindDefaultDataFile(defaultDataFileCandidates(cfg.SourceFile)); ok {
			cfg.Import.DataFile = detected
		}
	}

	if cfg.Import.DataFile != "" {
		cfg.Import.DataFile = resolvePathFromConfig(cfg.SourceFile, cfg.Import.DataFile)
	}

	return cfg, nil
}

func (c Config) DatabaseDSN() string {
	if strings.TrimSpace(c.Database.DSN) != "" {
		return strings.TrimSpace(c.Database.DSN)
	}

	switch c.Database.Driver {
	case "mysql":
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.Name,
			c.Database.Charset,
			boolToMySQL(c.Database.ParseTime),
			c.Database.Loc,
		)
	default:
		return c.Database.Name
	}
}

func defaultConfig() Config {
	return Config{
		App: AppConfig{
			Addr: ":8080",
		},
		Auth: AuthConfig{
			Issuer:                "brights-admin",
			JWTSecret:             "ChangeMeJWTSecret",
			AccessTokenTTLMinutes: 720,
		},
		Database: DatabaseConfig{
			Driver:             "mysql",
			Host:               "127.0.0.1",
			Port:               3306,
			Name:               "brights",
			Username:           "root",
			Password:           "ChangeMeMySQLPassword",
			Charset:            "utf8mb4",
			ParseTime:          true,
			Loc:                "Local",
			AutoCreateDatabase: true,
		},
		Bootstrap: BootstrapConfig{
			SuperAdmin: SuperAdminConfig{
				Enabled:     false,
				Username:    "",
				Password:    "",
				DisplayName: "",
			},
		},
		Import: ImportConfig{
			Enabled:     false,
			SubjectKey:  "english",
			Replace:     true,
			OnlyIfEmpty: true,
		},
	}
}

func (c *Config) normalize() {
	c.App.Addr = strings.TrimSpace(c.App.Addr)
	if c.App.Addr == "" {
		c.App.Addr = ":8080"
	}

	c.Auth.Issuer = strings.TrimSpace(c.Auth.Issuer)
	if c.Auth.Issuer == "" {
		c.Auth.Issuer = "brights-admin"
	}
	c.Auth.JWTSecret = strings.TrimSpace(c.Auth.JWTSecret)
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "ChangeMeJWTSecret"
	}
	if c.Auth.AccessTokenTTLMinutes <= 0 {
		c.Auth.AccessTokenTTLMinutes = 720
	}

	c.Database.Driver = strings.ToLower(strings.TrimSpace(c.Database.Driver))
	if c.Database.Driver == "" {
		c.Database.Driver = "mysql"
	}
	if c.Database.Port == 0 {
		if c.Database.Driver == "mysql" {
			c.Database.Port = 3306
		}
	}
	if strings.TrimSpace(c.Database.Host) == "" {
		c.Database.Host = "127.0.0.1"
	}
	if strings.TrimSpace(c.Database.Name) == "" {
		if c.Database.Driver == "mysql" {
			c.Database.Name = "brights"
		} else {
			c.Database.Name = filepath.Join(".", "brights.db")
		}
	}
	if strings.TrimSpace(c.Database.Charset) == "" {
		c.Database.Charset = "utf8mb4"
	}
	if strings.TrimSpace(c.Database.Loc) == "" {
		c.Database.Loc = "Local"
	}

	c.Bootstrap.SuperAdmin.Username = strings.TrimSpace(c.Bootstrap.SuperAdmin.Username)
	c.Bootstrap.SuperAdmin.Password = strings.TrimSpace(c.Bootstrap.SuperAdmin.Password)
	c.Bootstrap.SuperAdmin.DisplayName = strings.TrimSpace(c.Bootstrap.SuperAdmin.DisplayName)
	if c.Bootstrap.SuperAdmin.DisplayName == "" {
		c.Bootstrap.SuperAdmin.DisplayName = "超级管理员"
	}

	c.Import.SubjectKey = strings.TrimSpace(c.Import.SubjectKey)
	if c.Import.SubjectKey == "" {
		c.Import.SubjectKey = "english"
	}
	c.Import.DataFile = strings.TrimSpace(c.Import.DataFile)
}

func mergeJSONConfig(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config json %s: %w", path, err)
	}
	return nil
}

func resolveConfigPath(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Abs(strings.TrimSpace(explicit))
	}

	if fromEnv := strings.TrimSpace(os.Getenv("BRIGHTS_CONFIG_FILE")); fromEnv != "" {
		return filepath.Abs(fromEnv)
	}

	candidates := []string{
		filepath.Join(".", "configs", "app.json"),
		filepath.Join(".", "config.json"),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return filepath.Abs(candidate)
	}

	return "", nil
}

func resolvePathFromConfig(configFile, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if filepath.IsAbs(target) || configFile == "" {
		return target
	}
	baseDir := filepath.Dir(configFile)
	resolved := filepath.Join(baseDir, target)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return resolved
	}
	return abs
}

func applyEnvOverrides(cfg *Config) {
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_ADDR")); value != "" {
		cfg.App.Addr = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_AUTH_ISSUER")); value != "" {
		cfg.Auth.Issuer = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_AUTH_JWT_SECRET")); value != "" {
		cfg.Auth.JWTSecret = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_AUTH_ACCESS_TOKEN_TTL_MINUTES")); value != "" {
		if ttl, err := strconv.Atoi(value); err == nil {
			cfg.Auth.AccessTokenTTLMinutes = ttl
		}
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_DRIVER")); value != "" {
		cfg.Database.Driver = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_DSN")); value != "" {
		cfg.Database.DSN = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_HOST")); value != "" {
		cfg.Database.Host = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_NAME")); value != "" {
		cfg.Database.Name = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_USER")); value != "" {
		cfg.Database.Username = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_PASSWORD")); value != "" {
		cfg.Database.Password = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_CHARSET")); value != "" {
		cfg.Database.Charset = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_LOC")); value != "" {
		cfg.Database.Loc = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_PORT")); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			cfg.Database.Port = port
		}
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_PARSE_TIME")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Database.ParseTime = parsed
		}
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DB_AUTO_CREATE_DATABASE")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Database.AutoCreateDatabase = parsed
		}
	}

	if value := strings.TrimSpace(os.Getenv("BRIGHTS_BOOTSTRAP_ADMIN_ENABLED")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Bootstrap.SuperAdmin.Enabled = parsed
		}
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_BOOTSTRAP_ADMIN_USERNAME")); value != "" {
		cfg.Bootstrap.SuperAdmin.Username = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_BOOTSTRAP_ADMIN_PASSWORD")); value != "" {
		cfg.Bootstrap.SuperAdmin.Password = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_BOOTSTRAP_ADMIN_DISPLAY_NAME")); value != "" {
		cfg.Bootstrap.SuperAdmin.DisplayName = value
	}

	if value := strings.TrimSpace(os.Getenv("BRIGHTS_IMPORT_ENABLED")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Import.Enabled = parsed
		}
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_IMPORT_SUBJECT_KEY")); value != "" {
		cfg.Import.SubjectKey = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_DATA_FILE")); value != "" {
		cfg.Import.DataFile = value
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_IMPORT_REPLACE")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Import.Replace = parsed
		}
	}
	if value := strings.TrimSpace(os.Getenv("BRIGHTS_IMPORT_ONLY_IF_EMPTY")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Import.OnlyIfEmpty = parsed
		}
	}
}

func defaultDataFileCandidates(configFile string) []string {
	candidates := []string{
		filepath.Join("..", "brights_202605020108.csv"),
		filepath.Join("..", "brights_202605020108.xlsx"),
		"brights_202605020108.csv",
		"brights_202605020108.xlsx",
	}
	if configFile != "" {
		configDir := filepath.Dir(configFile)
		candidates = append(candidates,
			filepath.Join(configDir, "..", "..", "brights_202605020108.csv"),
			filepath.Join(configDir, "..", "..", "brights_202605020108.xlsx"),
		)
	}
	return candidates
}

func boolToMySQL(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func MustLoad(configPath string) Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(err)
	}
	return cfg
}
