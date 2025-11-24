package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

// RunMigrations executes embedded SQL migrations in lexical order.
func RunMigrations(ctx context.Context, db *sql.DB, migrations embed.FS) error {
	files, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		stmt, err := migrations.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}
		if len(stmt) == 0 {
			continue
		}
		if _, err := db.ExecContext(ctx, string(stmt)); err != nil {
			return fmt.Errorf("execute migration %s: %w", file, err)
		}
	}
	return nil
}
