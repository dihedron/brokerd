package store2

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
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"go.uber.org/zap"
)

// Cluster represents a raft cluster
type ReplicatedSQLiteStore struct {
	// RaftDirectory is the directory where all the Raft protocol files (e.g.
	// the log, the snaphots, the persistent store) will be kept.
	RaftDirectory string
	// RaftBindAddress is the network address on which the Raft protocol will
	// be listening on.
	RaftBindAddress string
	// RaftRetainSnapshotCount is the number of Raft snaphots to keep.
	RaftRetainSnapshotCount int
	// RaftTimeout is the timeout of the Raft cluster.
	RaftTimeout time.Duration
	// nodeID is the unique ID of the server in the cluster.
	nodeID string
	// raft is the underlying Raft consensus cluster.
	raft *raft.Raft
	// transport is the underlying transport layer for Raft.
	transport *raft.NetworkTransport
	// store is the actual SQLiteStore.
	store *SQLiteStore
}

// New creates a new Cluster and associates it with the given finite
// state machine (FSM), with the given cluster options.
func New(nodeID string, store *SQLiteStore, options ...Option) (*ReplicatedSQLiteStore, error) {

	if store == nil {
		log.L.Error("SQLite store must not be null")
		return nil, fmt.Errorf("invalid SQLite store reference")
	}

	c := &ReplicatedSQLiteStore{
		nodeID:                  nodeID,
		RaftDirectory:           "raft",
		RaftBindAddress:         "127.0.0.1:12000",
		RaftRetainSnapshotCount: DefaultRetainSnapshotCount,
		store:                   store,
	}
	// apply functional options
	for _, option := range options {
		option(c)
	}

	// setup Raft communication
	advertise, err := net.ResolveTCPAddr("tcp", c.RaftBindAddress)
	if err != nil {
		log.L.Error("error resolving bind address", zap.String("bind address", c.RaftBindAddress), zap.Error(err))
		return nil, err
	}
	transport, err := raft.NewTCPTransport(c.RaftBindAddress, advertise, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.L.Error("error creating Raft TCP transport", zap.String("bind address", c.RaftBindAddress), zap.Error(err))
		return nil, err
	}
	c.transport = transport

	// create the snapshot store; this allows the Raft to truncate the log
	snapshots, err := raft.NewFileSnapshotStore(c.RaftDirectory, DefaultRetainSnapshotCount, os.Stderr)
	if err != nil {
		log.L.Error("error creating file snaphost store", zap.String("directory", c.RaftDirectory), zap.Error(err))
		return nil, fmt.Errorf("file snapshot store: %s", err)
	}

	// create the underlying BoltDB store, used as both log store and stable store
	// var logStore raft.LogStore
	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(c.RaftDirectory, "raft.db"))
	if err != nil {
		return nil, fmt.Errorf("new bolt store: %s", err)
	}

	// instantiate the Raft systems
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	r, err := raft.NewRaft(config, c, boltDB, boltDB, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("new raft: %s", err)
	}
	c.raft = r
	return c, nil
}

// Node represents a node in the Cluster.
type Node struct {
	ID      string
	Address string
}

// Bootstrap bootstraps the Cluster with the set of nodes; if none is
// provided, the cluster is bootstrapped with this single node.
func (c *ReplicatedSQLiteStore) Bootstrap(nodes ...Node) error {
	configuration := raft.Configuration{
		Servers: []raft.Server{},
	}
	if len(nodes) == 0 {
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      raft.ServerID(c.nodeID),
			Address: c.transport.LocalAddr(),
		})
	} else {
		for _, node := range nodes {
			configuration.Servers = append(configuration.Servers, raft.Server{
				ID:      raft.ServerID(node.ID),
				Address: raft.ServerAddress(node.Address),
			})
		}
	}

	if f := c.raft.BootstrapCluster(configuration); f.Error() != nil {
		log.L.Error("error bootstrapping cluster", zap.Error(f.Error()))
		return f.Error()
	}
	log.L.Info("cluster bootstrapped successfully", zap.String("master node ID", c.nodeID))
	return nil
}

// Join joins a node, identified by nodeID and located at address, to
// this cluster. The node must be ready to respond to Raft communications
// at that address.
func (c *ReplicatedSQLiteStore) Join(nodeID string, address string) error {
	log.L.Info("received join request for remote node", zap.String("nodeID", nodeID), zap.String("address", address))

	configFuture := c.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.L.Error("failed to get raft configuration", zap.Error(err))
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(address) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(address) && srv.ID == raft.ServerID(nodeID) {
				log.L.Debug("node is already member of cluster, ignoring join request", zap.String("nodeID", nodeID), zap.String("address", address))
				return nil
			}

			future := c.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, address, err)
			}
		}
	}

	if f := c.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(address), 0, 0); f.Error() != nil {
		log.L.Error("error adding voter to cluster", zap.Error(f.Error()), zap.String("node ID", nodeID), zap.String("address", address))
		return f.Error()
	}
	log.L.Info("node joined successfully", zap.String("node ID", nodeID), zap.String("address", address))
	return nil
}

// Apply applies a Raft log entry to the key-value store.
func (f *ReplicatedSQLiteStore) Apply(l *raft.Log) interface{} {
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
func (r *ReplicatedSQLiteStore) Snapshot() (raft.FSMSnapshot, error) {

	// SQLite3 has a SERIALIZABLE isolation level by default;
	// in order to allow concurrent Apply() to proceed we declare
	// this transaction as ReadOnly.
	tx, err := r.store.db.BeginTx(context.Background(), &sql.TxOptions{
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

	return &SQLiteSnapshot{
		db:   r.store.db,
		tx:   tx,
		rows: rows,
	}, nil
}

// Restore stores the key-value store to a previous state.
func (f *ReplicatedSQLiteStore) Restore(rc io.ReadCloser) error {
	f.store.db.Begin()
	pairs := []pair{}
	if err := json.NewDecoder(rc).Decode(&pairs); err != nil {
		log.L.Error("error unmarshaling JSON to snapshot contents", zap.Error(err))
		return err
	}
	tx, err := f.store.db.BeginTx(context.Background(), &sql.TxOptions{
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

// CommandType represents the type of command.
type CommandType int8

const (
	// TypeGet is the "Get" command type.
	TypeGet CommandType = iota
	// TypeSet is the "Set" command type.
	TypeSet
	// TypeDelete is the "Delete" command type.
	TypeDelete
)

// Command is the Finite State Machine command.
type Command struct {
	Type  CommandType `json:"type"`
	Key   string      `json:"key"`
	Value string      `json:"value,omitempty"`
}

// Get retrieves the value corresponsing to the given key.
func (s *ReplicatedSQLiteStore) Get(key string) (string, error) {
	return s.store.Get(key)
}

// Get retrieves the value corresponsing to the given key.
// func (s *ReplicatedSQLiteStore) Get(key string) (string, error) {
// 	if s.cluster.Raft().State() != raft.Leader {
// 		err := fmt.Errorf("mutating operation not on Raft cluster leader")
// 		log.L.Error("error in Raft cluster operation", zap.Error(err))
// 		return "", err
// 	}

// 	b, err := json.Marshal(&Command{
// 		Type: TypeDelete,
// 		Key:  key,
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	f := s.cluster.Raft().Apply(b, s.cluster.RaftTimeout)
// 	if err := f.Error(); err != nil {
// 		log.L.Error("error in Raft cluster operation", zap.Error(err))
// 		return "", err
// 	}

// 	value, ok := f.Response().(string)
// 	if !ok {
// 		err := fmt.Errorf("get operation should return string, gor %T instead", f.Response())
// 		log.L.Error("invalid return type in cluster operation", zap.Error(err))
// 		return "", err
// 	}
// 	return value, nil
// }

// Set sets the value for the given key.
func (s *ReplicatedSQLiteStore) Set(key, value string) error {
	if s.cluster.Raft().State() != raft.Leader {
		err := fmt.Errorf("mutating operation not on Raft cluster leader")
		log.L.Error("error in Raft cluster operation", zap.Error(err))
		return err
	}

	b, err := json.Marshal(&Command{
		Type:  TypeSet,
		Key:   key,
		Value: value,
	})
	if err != nil {
		return err
	}

	f := s.cluster.Raft().Apply(b, s.cluster.RaftTimeout)
	return f.Error()
}

// Delete deletes the given key.
func (s *ReplicatedSQLiteStore) Delete(key string) error {
	if s.cluster.Raft().State() != raft.Leader {
		err := fmt.Errorf("mutating operation not on Raft cluster leader")
		log.L.Error("error in Raft cluster operation", zap.Error(err))
		return err
	}

	b, err := json.Marshal(&Command{
		Type: TypeDelete,
		Key:  key,
	})
	if err != nil {
		return err
	}

	f := s.cluster.Raft().Apply(b, s.cluster.RaftTimeout)
	return f.Error()
}
