package store4

import (
	"encoding/json"
	"fmt"

	"github.com/dihedron/brokerd/cluster"
	"github.com/dihedron/brokerd/log"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

type ReplicatedSQLiteStore struct {
	Store              *SQLiteStore
	Cluster            *cluster.Cluster
	AllowGetOnFollower bool
}

// Get retrieves the value corresponsing to the given key.
func (s *ReplicatedSQLiteStore) Get(key string) (string, error) {
	if s.AllowGetOnFollower {
		log.L.Debug("returning local value ")
		return s.Store.Get(key)
	} else {
		if s.Cluster.Raft.State() != raft.Leader {
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

		f := s.Cluster.Raft.Apply(b, s.Cluster.RaftTimeout)
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
	if s.Cluster.Raft.State() != raft.Leader {
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

	f := s.Cluster.Raft.Apply(b, s.Cluster.RaftTimeout)
	return f.Error()
}

// Delete deletes the given key.
func (s *ReplicatedSQLiteStore) Delete(key string) error {
	if s.Cluster.Raft.State() != raft.Leader {
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

	f := s.Cluster.Raft.Apply(b, s.Cluster.RaftTimeout)
	return f.Error()
}
