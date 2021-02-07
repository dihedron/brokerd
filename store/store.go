// Package store provides a simple distributed key-value store. The keys and
// associated values are changed via distributed consensus, meaning that the
// values are changed only when a majority of nodes in the cluster agree on
// the new value.
//
// Distributed consensus is provided via the Raft algorithm, specifically the
// Hashicorp implementation.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // load sqlite3 drivers

	"github.com/dihedron/brokerd/log"
	"github.com/dihedron/brokerd/migrations"
	"github.com/dihedron/brokerd/sqlite"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"go.uber.org/zap"
)

const (
	// DefaultRetainSnapshotCount is the default number of snaphots
	// to keep.
	DefaultRetainSnapshotCount = 2
	// DefaultRaftTimeout is the default timeout of the Raft cluster.
	DefaultRaftTimeout = 10 * time.Second
)

type command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Store is a simple key-value store, where all changes are made via Raft consensus.
type Store struct {
	// RaftDir is the directory where the Raft protocol files
	// will be kept.
	RaftDir string
	// RaftBind is the ...
	RaftBind string
	// db is the database for the FSM.
	db *sql.DB
	// raft is the Raft consensus mechanism.
	raft *raft.Raft

	// mu sync.Mutex
	// m  map[string]string // The key-value store for the system.
}

// New returns a new Store.
func New() *Store {
	return &Store{}
}

// Open opens the store. If enableSingle is set, and there are no existing peers,
// then this node becomes the first node, and therefore leader, of the cluster.
// localID should be the server identifier for this node.
func (s *Store) Open(enableSingle bool, localID string) error {
	// open the local database
	db, err := sqlite.InitDB(filepath.Join(s.RaftDir, "sqlite3.db"), migrations.Migrations)
	if err != nil {
		log.L.Error("error opening database", zap.Error(err))
		return err
	}
	// test the connection
	if err = db.Ping(); err != nil {
		log.L.Error("cannot ping database", zap.Error(err))
		return err
	}
	s.db = db
	log.L.Debug("database loaded")

	// setup Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(localID)

	// setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", s.RaftBind)
	if err != nil {
		return err
	}
	transport, err := raft.NewTCPTransport(s.RaftBind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	// create the snapshot store; this allows the Raft to truncate the log
	snapshots, err := raft.NewFileSnapshotStore(s.RaftDir, DefaultRetainSnapshotCount, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot store: %s", err)
	}

	// create the log store and stable store
	var logStore raft.LogStore
	var stableStore raft.StableStore
	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(s.RaftDir, "raft.db"))
	if err != nil {
		return fmt.Errorf("new bolt store: %s", err)
	}
	logStore = boltDB
	stableStore = boltDB

	// instantiate the Raft systems
	ra, err := raft.NewRaft(config, (*fsm)(s), logStore, stableStore, snapshots, transport)
	if err != nil {
		return fmt.Errorf("new raft: %s", err)
	}
	s.raft = ra

	if enableSingle {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		ra.BootstrapCluster(configuration)
	}

	return nil
}

// Get returns the value for the given key.
func (s *Store) Get(key string) (string, error) {

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
	return value, nil
	// s.mu.Lock()
	// defer s.mu.Unlock()
	// return s.m[key], nil
}

// Set sets the value for the given key.
func (s *Store) Set(key, value string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, DefaultRaftTimeout)
	return f.Error()
}

// Delete deletes the given key.
func (s *Store) Delete(key string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:  "delete",
		Key: key,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, DefaultRaftTimeout)
	return f.Error()
}

// Join joins a node, identified by nodeID and located at addr, to this store.
// The node must be ready to respond to Raft communications at that address.
func (s *Store) Join(nodeID, addr string) error {
	log.L.Info("received join request for remote node", zap.String("nodeID", nodeID), zap.String("address", addr))

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.L.Error("failed to get raft configuration", zap.Error(err))
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				log.L.Debug("node is already member of cluster, ignoring join request", zap.String("nodeID", nodeID), zap.String("address", addr))
				return nil
			}

			future := s.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	log.L.Info("node joined successfully", zap.String("nodeID", nodeID), zap.String("address", addr))
	return nil
}

type fsm Store

// Apply applies a Raft log entry to the key-value store.
func (f *fsm) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	switch c.Op {
	case "set":
		return f.applySet(c.Key, c.Value)
	case "delete":
		return f.applyDelete(c.Key)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", c.Op))
	}
}

// Snapshot returns a snapshot of the key-value store.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {

	// SQLite3 has a SERIALIZABLE isolation level by default;
	// in order to allow concurrent Apply() to proceed we declare
	// this transaction as ReadOnly.
	tx, err := f.db.BeginTx(context.Background(), &sql.TxOptions{
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

	return &fsmSnapshot{
		db:   f.db,
		tx:   tx,
		rows: rows,
	}, nil
}

// Restore stores the key-value store to a previous state.
func (f *fsm) Restore(rc io.ReadCloser) error {
	// o := make(map[string]string)
	f.db.Begin()
	pairs := []pair{}
	if err := json.NewDecoder(rc).Decode(&pairs); err != nil {
		log.L.Error("error unmarshaling JSON to snapshot contents", zap.Error(err))
		return err
	}
	tx, err := f.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}

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

	// Set the state from the snapshot, no lock required according to
	// Hashicorp docs.
	// f.m = o
	return nil
}

func (f *fsm) applySet(key, value string) interface{} {
	tx, err := f.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	_, err = f.db.Exec("INSERT OR REPLACE INTO pairs (key,value) VALUES (?,?)", key, value)
	if err != nil {
		log.L.Error("error inserting value into database", zap.String("key", key), zap.String("value", value), zap.Error(err))
		return err
	}
	tx.Commit()
	log.L.Debug("value stored into database", zap.String("key", key), zap.String("value", value))
	return nil
}

func (f *fsm) applyDelete(key string) interface{} {
	tx, err := f.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		log.L.Error("error opening transaction", zap.Error(err))
		return err
	}
	_, err = f.db.Exec("DELETE FROM pairs where key=?", key)
	if err != nil {
		log.L.Error("error deleting pair", zap.String("key", key), zap.Error(err))
		return err
	}
	tx.Commit()
	log.L.Debug("value deleted from database", zap.String("key", key))
	return nil
}
