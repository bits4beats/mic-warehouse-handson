package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"warehouse.local/core/repositories"
)

func main() {
	legacy, warehouse, mode, err := wireRepositories()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	defer legacy.Close()
	defer warehouse.Close()

	dual := repositories.NewDualWriteArticleRepository(
		repositories.NewLegacyMySQLArticleRepository(legacy),
		repositories.NewMySQLArticleRepository(warehouse),
		mode,
	)
	// Phase 04 exposes only /health; the dual-write repository is wired here so the
	// composition root is complete. The seed CLI (cmd/seed) exercises it against the
	// two real databases; HTTP handlers arrive in Phase 06.
	_ = dual

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]any{
			"status": "ok",
			"mode":   readModeName(mode),
		})
	})

	log.Printf("Starting server on :8081 (read mode: %s)", readModeName(mode))
	e.Logger.Fatal(e.Start(":8081"))
}

// wireRepositories opens the legacy and warehouse MySQL connections from env vars.
// Falls back to localhost defaults so unit-test code that imports main does not break;
// at runtime the docker-compose env values are used.
func wireRepositories() (*sql.DB, *sql.DB, repositories.ReadMode, error) {
	legacyDSN := buildDSN(
		envOr("LEGACY_DB_USER", "legacy_user"),
		envOr("LEGACY_DB_PASSWORD", "legacy_pass"),
		envOr("LEGACY_DB_HOST", "127.0.0.1"),
		envOr("LEGACY_DB_PORT", "3306"),
		envOr("LEGACY_DB_NAME", "legacy_db"),
	)
	warehouseDSN := buildDSN(
		envOr("WAREHOUSE_DB_USER", "warehouse_user"),
		envOr("WAREHOUSE_DB_PASSWORD", "warehouse_pass"),
		envOr("WAREHOUSE_DB_HOST", "127.0.0.1"),
		envOr("WAREHOUSE_DB_PORT", "3307"),
		envOr("WAREHOUSE_DB_NAME", "warehouse_db"),
	)
	legacy, err := openDB(legacyDSN)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("open legacy db: %w", err)
	}
	warehouse, err := openDB(warehouseDSN)
	if err != nil {
		legacy.Close()
		return nil, nil, 0, fmt.Errorf("open warehouse db: %w", err)
	}

	mode, err := parseReadMode(envOr("DUAL_WRITE_READ_MODE", "legacy"))
	if err != nil {
		legacy.Close()
		warehouse.Close()
		return nil, nil, 0, err
	}

	return legacy, warehouse, mode, nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func buildDSN(user, pass, host, port, name string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
		user, pass, host, port, name)
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseReadMode(s string) (repositories.ReadMode, error) {
	switch strings.ToLower(s) {
	case "legacy":
		return repositories.ReadFromLegacy, nil
	case "bc":
		return repositories.ReadFromBC, nil
	default:
		return 0, errors.New("DUAL_WRITE_READ_MODE must be legacy|bc")
	}
}

func readModeName(m repositories.ReadMode) string {
	switch m {
	case repositories.ReadFromLegacy:
		return "legacy"
	case repositories.ReadFromBC:
		return "bc"
	default:
		return "unknown"
	}
}
