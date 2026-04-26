package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"git.romanzipp.net/romanzipp/news/internal/config"
)

func Open(cfg *config.Config) (*sql.DB, error) {
	switch cfg.DBDriver {
	case "sqlite":
		dsn := cfg.DBDsn
		if strings.HasPrefix(dsn, "file:") {
			path := strings.SplitN(dsn, "?", 2)[0]
			path = strings.TrimPrefix(path, "file:")
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("create data dir: %w", err)
			}
		}
		return sql.Open("sqlite3", dsn)
	case "postgres":
		return sql.Open("postgres", cfg.DBDsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DBDriver)
	}
}

func Migrate(db *sql.DB, driver string) error {
	for _, stmt := range migrations(driver) {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}
