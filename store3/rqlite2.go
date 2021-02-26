package store3

import (
	"encoding/json"
	"fmt"

	"github.com/dihedron/brokerd/cluster"
	"github.com/dihedron/brokerd/log"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

// ReplicatedSQLiteStore provides Raft-base replication
// of the SQLiteStore across the cluster.
type ReplicatedSQLiteStore struct {
	// allowGetOnFollower enables the bypass of the cluster
	// consensus for idempotent operations; may return stale
	// data while the the follower is applying the Raft log
	// entries.
	allowGetOnFollower bool
	// store is the actual SQLiteStore.
	store *SQLiteStore
	// cluster is the Raft consensus mechanism.
	cluster *cluster.Cluster
}

// NewReplicatedSQLiteStore creates and initialises a new SQLite-based store.
func NewReplicatedSQLiteStore(store *SQLiteStore) (*ReplicatedSQLiteStore, error) {
	if store == nil {
		log.L.Error("SQLite store must not be null")
		return nil, fmt.Errorf("invalid SQLite store reference")
	}
	// if cluster == nil {
	// 	log.L.Error("Raft cluster must not be null")
	// 	return nil, fmt.Errorf("invalid Raft cluster store reference")
	// }
	return &ReplicatedSQLiteStore{
		//allowGetOnFollower: allowGetOnFollower,
		store: store,
		// cluster:            cluster,
	}, nil
}

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

// Get retrieves the value corresponsing to the given key.
func (s *ReplicatedSQLiteStore) Get(key string) (string, error) {
	if s.allowGetOnFollower {
		log.L.Debug("returning local value ")
		return s.store.Get(key)
	} else {
		if s.cluster.Raft.State() != raft.Leader {
			err := fmt.Errorf("getting value from Raft cluster followers is not enabled in this store")
			log.L.Error("error in Raft cluster operation", zap.Error(err))
			return "", err
		}
		b, err := json.Marshal(&Command{
			Type: Delete,
			Key:  key,
		})
		if err != nil {
			return "", err
		}

		f := s.cluster.Raft.Apply(b, s.cluster.RaftTimeout)
		if err := f.Error(); err != nil {
			log.L.Error("error in Raft cluster operation", zap.Error(err))
			return "", err
		}

		value, ok := f.Response().(string)
		if !ok {
			err := fmt.Errorf("get operation should return string, gor %T instead", f.Response())
			log.L.Error("invalid return type in cluster operation", zap.Error(err))
			return "", err
		}
		return value, nil
	}
}

// Set sets the value for the given key.
func (s *ReplicatedSQLiteStore) Set(key, value string) error {
	if s.cluster.Raft.State() != raft.Leader {
		err := fmt.Errorf("mutating operation not on Raft cluster leader")
		log.L.Error("error in Raft cluster operation", zap.Error(err))
		return err
	}

	b, err := json.Marshal(&Command{
		Type:  Set,
		Key:   key,
		Value: value,
	})
	if err != nil {
		return err
	}

	f := s.cluster.Raft.Apply(b, s.cluster.RaftTimeout)
	return f.Error()
}

// Delete deletes the given key.
func (s *ReplicatedSQLiteStore) Delete(key string) error {
	if s.cluster.Raft.State() != raft.Leader {
		err := fmt.Errorf("mutating operation not on Raft cluster leader")
		log.L.Error("error in Raft cluster operation", zap.Error(err))
		return err
	}

	b, err := json.Marshal(&Command{
		Type: Delete,
		Key:  key,
	})
	if err != nil {
		return err
	}

	f := s.cluster.Raft.Apply(b, s.cluster.RaftTimeout)
	return f.Error()
}
