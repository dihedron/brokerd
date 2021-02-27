package kvstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"

	"github.com/dihedron/brokerd/log"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

// CommandType represents the type of command.
type CommandType int8

const (
	// Get is the "Get" command type.
	Get CommandType = iota
	// Set is the "Set" command type.
	Set
	// Delete is the "Delete" command type.
	Delete
)

// Command is the Finite State Machine command.
type Command struct {
	Type  CommandType `json:"type"`
	Key   string      `json:"key"`
	Value string      `json:"value,omitempty"`
}

// ReplicatedStoreFSM is an SQLite-base Raft Finite State Machine.
type ReplicatedStoreFSM struct {
	store *LocalStore
}

func NewReplicatedStoreFSM(store *LocalStore) *ReplicatedStoreFSM {
	return &ReplicatedStoreFSM{
		store: store,
	}
}

// Apply log is invoked once a log entry is committed.
// It returns a value which will be made available in the
// ApplyFuture returned by Raft.Apply method if that
// method was called on the same Raft node as the FSM.
func (s *ReplicatedStoreFSM) Apply(l *raft.Log) interface{} {
	var command Command
	if err := json.Unmarshal(l.Data, &command); err != nil {
		log.L.Error("failed to marshal command", zap.Error(err))
		return err
	}
	// NOTE: Get does not MUTATE the FSM, thus it needs not
	// go though the FSM.Apply rigmarole; it can be served directly
	// from the local store; this is handled in the RemoteStore.
	switch command.Type {
	case Set:
		err := s.store.Set(command.Key, command.Value)
		if err != nil {
			log.L.Error("error retrieving value from SQLite store", zap.String("key", command.Key), zap.Error(err))
			return err
		}
		log.L.Debug("value stored", zap.String("key", command.Key), zap.String("value", command.Value))
		return nil
	case Delete:
		err := s.store.Delete(command.Key)
		if err != nil {
			log.L.Error("error retrieving value from SQLite store", zap.String("key", command.Key), zap.Error(err))
			return err
		}
		log.L.Debug("value deleted", zap.String("key", command.Key))
		return nil
	default:
		err := fmt.Errorf("unrecognized command op: %d", command.Type)
		log.L.Error("failure applying log entry", zap.Error(err))
		return err
	}
}

// Snapshot returns a snapshot of the key-value store, to support
// log compaction; the returned ReplicatedStoreFSM...
func (s *ReplicatedStoreFSM) Snapshot() (raft.FSMSnapshot, error) {
	// SQLite3 has a SERIALIZABLE isolation level by default;
	// in order to allow concurrent Apply() to proceed we declare
	// this transaction as ReadOnly.
	tx, err := s.store.DB.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  true,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return nil, err
	}
	// run the query now and keep the cursor open
	rows, err := tx.Query("SELECT key, value FROM pairs")
	if err != nil {
		log.L.Error("error running query", zap.Error(err))
		tx.Rollback()
		return nil, err
	}

	return &SQLiteFSMSnapshot{
		db:   s.store.DB,
		tx:   tx,
		rows: rows,
	}, nil
}

// Restore restores the FSM to a previous state from a snapshot.
func (s *ReplicatedStoreFSM) Restore(data io.ReadCloser) error {
	s.store.DB.Begin()
	pairs := []pair{}
	if err := json.NewDecoder(data).Decode(&pairs); err != nil {
		log.L.Error("error unmarshaling JSON to snapshot contents", zap.Error(err))
		return err
	}
	tx, err := s.store.DB.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	// the FSM state must be discarded prior to restoring
	if _, err := tx.Exec("DELETE FROM pairs;"); err != nil {
		log.L.Error("error truncating table")
		tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO pairs (key,value) VALUES (?,?)")
	if err != nil {
		log.L.Error("error preparing insert statement", zap.Error(err))
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, pair := range pairs {
		if _, err = stmt.Exec(pair.Key, pair.Value); err != nil {
			log.L.Error("error executing statement", zap.Error(err))
			break
		}
	}
	if err != nil {
		log.L.Error("error restoring snaphot", zap.Error(err))
		tx.Rollback()
	}
	log.L.Debug("restore complete, committing transaction")
	tx.Commit()
	return nil
}
