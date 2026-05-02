package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"brights/api/internal/bootstrap"
	"brights/api/internal/config"
	"brights/api/internal/service"
	"brights/api/internal/storage"
)

func main() {
	configPath := flag.String("config", "", "path to config json file")
	modeFlag := flag.String("mode", "setup", "bootstrap mode: setup or reset")
	usernameFlag := flag.String("username", "", "super admin username")
	passwordFlag := flag.String("password", "", "super admin password")
	displayNameFlag := flag.String("display-name", "", "super admin display name")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	username := defaultValue(*usernameFlag, cfg.Bootstrap.SuperAdmin.Username)
	password := defaultValue(*passwordFlag, cfg.Bootstrap.SuperAdmin.Password)
	displayName := defaultValue(*displayNameFlag, cfg.Bootstrap.SuperAdmin.DisplayName)

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

	switch strings.ToLower(strings.TrimSpace(*modeFlag)) {
	case "", "setup":
		admin, created, err := bootstrap.SuperAdmin(context.Background(), svc, username, password, displayName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("super admin ready: username=%s created=%t\n", admin.Username, created)
	case "reset":
		admin, err := bootstrap.ResetSuperAdminPassword(context.Background(), svc, username, password, displayName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("super admin password reset: username=%s\n", admin.Username)
	default:
		log.Fatalf("unsupported mode: %s", *modeFlag)
	}
}

func defaultValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
