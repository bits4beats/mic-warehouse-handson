package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	// Read-source feature toggle. A running process's env is immutable, so the live
	// toggle is backed by a file (DUAL_WRITE_READ_MODE_FILE): edit it, then SIGHUP the
	// process to flip legacy<->bc without a restart. Mount it via a volume/ConfigMap.
	// Send with `kill -HUP <pid>` or, in a container, `docker compose kill -s HUP app`.
	// If the file is absent the DUAL_WRITE_READ_MODE env value (used at startup) stands.
	watchReadModeToggle(dual)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]any{
			"status": "ok",
			"mode":   readModeName(dual.ReadMode()),
		})
	})

	log.Printf("Starting server on :8081 (read mode: %s)", readModeName(dual.ReadMode()))
	e.Logger.Fatal(e.Start(":8081"))
}

// watchReadModeToggle reloads the read-source toggle from the file named by
// DUAL_WRITE_READ_MODE_FILE whenever the process receives SIGHUP, letting operators
// switch legacy<->bc during migration without a restart. A missing file or invalid
// value is logged and ignored, so an operator mistake cannot take the service down.
func watchReadModeToggle(dual *repositories.DualWriteArticleRepository) {
	path := os.Getenv("DUAL_WRITE_READ_MODE_FILE")
	if path == "" {
		log.Print("read-mode toggle: DUAL_WRITE_READ_MODE_FILE unset, SIGHUP reload disabled")
		return
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for range sig {
			mode, err := readModeFromFile(path)
			if err != nil {
				log.Printf("SIGHUP: keeping read mode %s (%v)", readModeName(dual.ReadMode()), err)
				continue
			}
			dual.SetReadMode(mode)
			log.Printf("SIGHUP: read mode set to %s (from %s)", readModeName(mode), path)
		}
	}()
}

// readModeFromFile reads and parses the read-mode toggle file (its whole content is
// "legacy" or "bc", surrounding whitespace ignored).
func readModeFromFile(path string) (repositories.ReadMode, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	return parseReadMode(string(raw))
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
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "legacy":
		return repositories.ReadFromLegacy, nil
	case "bc":
		return repositories.ReadFromBC, nil
	default:
		return 0, fmt.Errorf("read mode must be legacy|bc, got %q", strings.TrimSpace(s))
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
