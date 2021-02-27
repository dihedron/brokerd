package kvstore

import (
	"context"
	"database/sql"

	"github.com/dihedron/brokerd/log"
	"github.com/dihedron/brokerd/sqlite"
	"go.uber.org/zap"
)

// LocalStore is the non-replicated, local-only SQLite-based implementation
// of the KVStore interface.
type LocalStore struct {
	sqlite.Store
}

// NewLocalStore creates a new SQLite-based, non-replicated implementation
// of the KVSTore interface.
func NewLocalStore(options ...sqlite.Option) (*LocalStore, error) {
	store, err := sqlite.New(options...)
	if err != nil {
		log.L.Error("error allocating base SQLite store", zap.Error(err))
		return nil, err
	}
	return &LocalStore{
		*store,
	}, nil
}

// Get returns the value for the given key.
func (s *LocalStore) Get(key string) (string, error) {
	tx, err := s.DB.BeginTx(context.Background(), &sql.TxOptions{
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
func (s *LocalStore) Set(key string, value string) error {
	tx, err := s.DB.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	_, err = s.DB.Exec("INSERT OR REPLACE INTO pairs (key,value) VALUES (?,?)", key, value)
	if err != nil {
		log.L.Error("error inserting value into database", zap.String("key", key), zap.String("value", value), zap.Error(err))
		return err
	}
	tx.Commit()
	log.L.Debug("value stored into database", zap.String("key", key), zap.String("value", value))
	return nil
}

// Delete removes the key/value pair from the store.
func (s *LocalStore) Delete(key string) error {
	tx, err := s.DB.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	_, err = s.DB.Exec("DELETE FROM pairs where key=?", key)
	if err != nil {
		log.L.Error("error deleting pair", zap.String("key", key), zap.Error(err))
		return err
	}
	tx.Commit()
	log.L.Debug("value deleted from database", zap.String("key", key))
	return nil
}
