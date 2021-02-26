package store3

import (
	"context"
	"database/sql"
	"path/filepath"

	"github.com/dihedron/brokerd/log"
	"github.com/dihedron/brokerd/migrations"
	"github.com/dihedron/brokerd/sqlite"
	"go.uber.org/zap"
)

const (
	// DefaultSQLite3FileName is the default name for the SQLite3
	// on-disk data file.
	DefaultSQLite3FileName string = "sqlite3.db"
)

// SQLiteStore is a store based on a local SQLite database.
type SQLiteStore struct {
	// db is the database for the FSM.
	db *sql.DB
}

// SQLiteStoreOptions contains all configuration parameters for
// the SQLiteStore.
type SQLiteStoreOptions struct {
	// StoreDirectory is the directory whethe the DB files are stored.
	StoreDirectory string
	// StoreFileName is the name of the
	StoreFileName string
}

// NewSQLiteStore creates and initialises a new SQLite-based store.
func NewSQLiteStore(options *SQLiteStoreOptions) (*SQLiteStore, error) {
	store := &SQLiteStore{}
	// open the local database
	db, err := sqlite.InitDB(filepath.Join(options.StoreDirectory, DefaultSQLite3FileName), migrations.Migrations)
	if err != nil {
		log.L.Error("error opening database", zap.Error(err))
		return nil, err
	}
	// test the connection
	if err = db.Ping(); err != nil {
		log.L.Error("cannot ping database", zap.Error(err))
		return nil, err
	}
	store.db = db
	log.L.Debug("database loaded")
	return store, nil
}

// Get returns the value for the given key.
func (s *SQLiteStore) Get(key string) (string, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  true,
	})
	if err != nil {
		log.L.Error("error opening read-only transaction", zap.Error(err))
		tx.Rollback()
		return "", err
	}
	value := ""
	if err := tx.QueryRow("SELECT value FROM pairs WHERE key=?", key).Scan(&value); err != nil {
		log.L.Error("error querying row", zap.String("key", key), zap.Error(err))
		tx.Rollback()
		return "", err
	}
	tx.Commit()
	log.L.Debug("returning value", zap.String("key", key), zap.String("value", value))
	return value, nil
}

// Set sets a value under the given key; if existing, it is updated,
// otherwise a new key/value pair is created.
func (s *SQLiteStore) Set(key string, value string) error {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	_, err = s.db.Exec("INSERT OR REPLACE INTO pairs (key,value) VALUES (?,?)", key, value)
	if err != nil {
		log.L.Error("error inserting value into database", zap.String("key", key), zap.String("value", value), zap.Error(err))
		return err
	}
	tx.Commit()
	log.L.Debug("value stored into database", zap.String("key", key), zap.String("value", value))
	return nil
}

// Delete removes the key/value pair from the store.
func (s *SQLiteStore) Delete(key string) error {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	_, err = s.db.Exec("DELETE FROM pairs where key=?", key)
	if err != nil {
		log.L.Error("error deleting pair", zap.String("key", key), zap.Error(err))
		return err
	}
	tx.Commit()
	log.L.Debug("value deleted from database", zap.String("key", key))
	return nil
}
