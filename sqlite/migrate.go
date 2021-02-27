package sqlite

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"

	_ "github.com/mattn/go-sqlite3" // SQLite3 drivers

	"github.com/dihedron/brokerd/log"
	"go.uber.org/zap"
)

// migrate sets up migration tracking and executes pending migration files.
//
// Migration files can be on disk or embedded in the executable. This function
// gets a list and sorts it so that they are executed in lexigraphical order.
//
// Once a migration is run, its name is stored in the 'migrations' table so it
// is not re-executed. Migrations run in a transaction to prevent partial
// migrations.
func migrate(db *sql.DB, migrations fs.FS) error {
	log.L.Debug("applying migrations...")

	// ensure the 'migrations' table exists so we don't duplicate migrations.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY);`); err != nil {
		err = fmt.Errorf("cannot create migrations table: %w", err)
		log.L.Error("error creating migrations table", zap.Error(err))
		return err
	}

	// read migration files from our embedded file system;
	// this uses Go 1.16's 'embed' package.
	names, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		log.L.Error("error getting list of migrations", zap.Error(err))
		return err
	}
	sort.Strings(names)

	// loop over all migration files and execute them in order
	for _, name := range names {
		if err := migrateFile(db, migrations, name); err != nil {
			err = fmt.Errorf("migration error: name=%q err=%w", name, err)
			log.L.Error("error applying migration file", zap.String("name", name), zap.Error(err))
			return err
		}
	}
	log.L.Debug("all migrations applied")
	return nil
}

// migrate runs a single migration file within a transaction; on success, the
// migration file name is saved to the "migrations" table to prevent re-running.
func migrateFile(db *sql.DB, migrations fs.FS, name string) error {
	log.L.Debug("applying migration file", zap.String("name", name))
	tx, err := db.Begin()
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// ensure migration has not already been run
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM migrations WHERE name = ?`, name).Scan(&n); err != nil {
		log.L.Error("error reading migrations table", zap.String("name", name), zap.Error(err))
		return err
	} else if n != 0 {
		log.L.Debug("migration already applied, skipping", zap.String("name", name))
		return nil // already run migration, skip
	}

	// read and execute migration file
	if buffer, err := fs.ReadFile(migrations, name); err != nil {
		log.L.Error("error reading migration file", zap.String("name", name), zap.Error(err))
		return err
	} else if _, err := tx.Exec(string(buffer)); err != nil {
		log.L.Error("error executing migration command", zap.String("command", string(buffer)), zap.Error(err))
		return err
	}

	// insert record into migrations to prevent re-running migration
	if _, err := tx.Exec(`INSERT INTO migrations (name) VALUES (?)`, name); err != nil {
		log.L.Error("error inserting migration into migrations table", zap.String("name", name))
		return err
	}
	log.L.Debug("migration applied", zap.String("name", name))
	return tx.Commit()
}
