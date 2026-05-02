package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"brights/api/internal/config"
	"brights/api/internal/domain"
	"brights/api/internal/service"
	"brights/api/internal/storage"
)

func main() {
	configPath := flag.String("config", "", "path to config json file")
	pathFlag := flag.String("path", "", "word data file path")
	subjectFlag := flag.String("subject", "", "subject key")
	replaceFlag := flag.String("replace", "", "replace existing words: true/false")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	dataFile := defaultValue(*pathFlag, cfg.Import.DataFile)
	subjectKey := defaultValue(*subjectFlag, cfg.Import.SubjectKey)
	replace := cfg.Import.Replace
	if strings.TrimSpace(*replaceFlag) != "" {
		parsed, err := parseBool(*replaceFlag)
		if err != nil {
			log.Fatal(err)
		}
		replace = parsed
	}

	if strings.TrimSpace(dataFile) == "" {
		log.Fatal("data file path is required")
	}

	db, err := storage.Open(cfg.Database.Driver, cfg.DatabaseDSN(), cfg.Database.AutoCreateDatabase)
	if err != nil {
		log.Fatal(err)
	}
	if err := storage.MigrateLegacySchema(db, cfg.Database.Driver); err != nil {
		log.Fatal(err)
	}
	if err := storage.AutoMigrate(db); err != nil {
		log.Fatal(err)
	}

	svc := service.New(db)
	if err := svc.SeedDefaults(context.Background()); err != nil {
		log.Fatal(err)
	}

	result, err := svc.ImportWordsFromFile(context.Background(), domain.ImportWordsInput{
		Path:       dataFile,
		SubjectKey: subjectKey,
		Replace:    boolPtr(replace),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"word import completed: imported=%d created_categories=%d subject=%s path=%s replace=%t\n",
		result.ImportedCount,
		result.CreatedCategories,
		result.SubjectKey,
		result.Path,
		result.Replace,
	)
}

func defaultValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y":
		return true, nil
	case "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", value)
	}
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
