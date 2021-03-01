package sqlite

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dihedron/brokerd/log"
	"github.com/dihedron/brokerd/migrations"
	"go.uber.org/zap"
)

const (
	// DefaultSQLite3FileName is the default name for the SQLite3
	// on-disk data file.
	DefaultSQLite3FileName string = "sqlite3.db"
)

// Store is a store based on a local SQLite database.
type Store struct {
	// db is the database for the FSM.
	DB *sql.DB
	// DataDirectory is the directory where the DB files are stored.
	DataDirectory string
	// DataFileName is the name of the SQLite3 data file.
	DataFileName string
}

// New creates and initialises a new SQLite-based store.
func New(options ...Option) (*Store, error) {
	// setup with defaults
	store := &Store{
		DataDirectory: DefaultStoreDirectory,
		DataFileName:  DefaultStoreFileName,
	}
	// apply functional options to override
	for _, option := range options {
		option(store)
	}
	log.L.Debug("creating SQLite3 database", zap.String("data directory", store.DataDirectory), zap.String("data file name", store.DataFileName))
	// open the local database
	db, err := initialise(filepath.Join(store.DataDirectory, store.DataFileName), migrations.Migrations)
	if err != nil {
		log.L.Error("error opening database", zap.Error(err))
		return nil, err
	}
	// test the connection
	if err = db.Ping(); err != nil {
		log.L.Error("cannot ping database", zap.Error(err))
		return nil, err
	}
	store.DB = db
	log.L.Debug("database loaded")
	return store, nil
}

// initialise opens and initialises an SQLite3 DB with all
// correct settings.
func initialise(dsn string, migrations fs.FS) (db *sql.DB, err error) {
	// ensure a DSN is set before attempting to open the database
	if dsn == "" {
		err = fmt.Errorf("dsn required")
		log.L.Error("the database DSN must be specified", zap.Error(err))
		return
	}
	log.L.Debug("opening SQLite3 database", zap.String("DSN", dsn))
	// make the parent directory unless using an in-memory db
	if dsn != ":memory:" {
		if err = os.MkdirAll(filepath.Dir(dsn), 0700); err != nil {
			log.L.Error("error creating directory for on-disk DB file", zap.String("path", dsn), zap.Error(err))
			return
		}
	}
	// open the database
	if db, err = sql.Open("sqlite3", dsn); err != nil {
		log.L.Error("error connecting to the database", zap.Error(err))
		return
	}
	// enable WAL; SQLite performs better with the WAL because it allows
	// multiple readers to operate while data is being written
	if _, err = db.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		err = fmt.Errorf("enable wal: %w", err)
		log.L.Error("error enabling WAL", zap.Error(err))
		return
	}
	// enable foreign key checks: for historical reasons, SQLite does not check
	// foreign key constraints by default... which is kinda insane; there's some
	// overhead on inserts to verify foreign key integrity but it's definitely
	// worth it.
	if _, err = db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		err = fmt.Errorf("foreign keys pragma: %w", err)
		log.L.Error("error enabling foreign keys checks", zap.Error(err))
		return
	}
	// apply migrations (if any)
	if err = migrate(db, migrations); err != nil {
		err = fmt.Errorf("migrate: %w", err)
		log.L.Error("error applying migrations", zap.Error(err))
		return
	}
	return
}
