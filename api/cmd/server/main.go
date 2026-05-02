package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"brights/api/internal/adminauth"
	"brights/api/internal/bootstrap"
	"brights/api/internal/config"
	"brights/api/internal/domain"
	"brights/api/internal/httpapi"
	"brights/api/internal/service"
	"brights/api/internal/storage"
	"brights/api/internal/userauth"
)

func main() {
	configPath := flag.String("config", "", "path to config json file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

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
	if err := svc.SeedDefaults(ctx); err != nil {
		log.Fatal(err)
	}
	authManager := adminauth.NewManager(
		cfg.Auth.Issuer,
		cfg.Auth.JWTSecret,
		time.Duration(cfg.Auth.AccessTokenTTLMinutes)*time.Minute,
	)
	userAuthManager := userauth.NewManager(
		cfg.Auth.Issuer,
		cfg.Auth.JWTSecret,
		time.Duration(cfg.Auth.AccessTokenTTLMinutes)*time.Minute,
	)

	if cfg.Bootstrap.SuperAdmin.Enabled &&
		strings.TrimSpace(cfg.Bootstrap.SuperAdmin.Username) != "" &&
		strings.TrimSpace(cfg.Bootstrap.SuperAdmin.Password) != "" {
		admin, created, err := bootstrap.SuperAdmin(
			ctx,
			svc,
			cfg.Bootstrap.SuperAdmin.Username,
			cfg.Bootstrap.SuperAdmin.Password,
			cfg.Bootstrap.SuperAdmin.DisplayName,
		)
		if err != nil {
			log.Fatalf("bootstrap super admin failed: %v", err)
		}
		log.Printf("super admin ready: username=%s created=%t", admin.Username, created)
	} else {
		setupStatus, setupErr := svc.GetAdminSetupStatus(ctx)
		if setupErr != nil {
			log.Printf("check admin setup status failed: %v", setupErr)
		} else if !setupStatus.Initialized {
			log.Printf("no admin initialized yet, visit /admin to complete first-run super admin setup")
		}
	}

	if cfg.Import.Enabled && strings.TrimSpace(cfg.Import.DataFile) != "" {
		if cfg.Import.OnlyIfEmpty {
			result, imported, err := svc.EnsureInitialImport(ctx, cfg.Import.SubjectKey, cfg.Import.DataFile)
			if err != nil {
				log.Printf("initial import skipped with error: %v", err)
			} else if imported {
				log.Printf("imported %d words into database from %s", result.ImportedCount, result.Path)
			} else {
				log.Printf("database already has words, initial import skipped")
			}
		} else {
			result, err := svc.ImportWordsFromFile(ctx, domain.ImportWordsInput{
				Path:       cfg.Import.DataFile,
				SubjectKey: cfg.Import.SubjectKey,
				Replace:    boolPtr(cfg.Import.Replace),
			})
			if err != nil {
				log.Printf("import on start failed: %v", err)
			} else {
				log.Printf("imported %d words into database from %s", result.ImportedCount, result.Path)
			}
		}
	}

	server := httpapi.NewServer(svc, authManager, userAuthManager)
	log.Printf("brights api listening on %s", cfg.App.Addr)
	if err := http.ListenAndServe(cfg.App.Addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
