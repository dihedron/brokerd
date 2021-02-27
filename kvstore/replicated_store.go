package kvstore

import (
	"encoding/json"
	"fmt"

	"github.com/dihedron/brokerd/cluster"
	"github.com/dihedron/brokerd/log"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

var (
	// ErrNotLeader is the error returned whn an operation that is
	// only permitted on the leader is attempted on a follower node.
	ErrNotLeader error = fmt.Errorf("operation not permitted on followers")
)

// ReplicatedStore is the replicated, Raft-based implementation
// of the KVStore interface; it achieves this by coordinating the
// Raft cluster and the local database.
type ReplicatedStore struct {
	store              *LocalStore
	cluster            *cluster.Cluster
	allowGetOnFollower bool
}

// NewReplicatedStore allocates a new ReplicatedStore which will
// funnel the commands through the Raft log, leaving it to the
// cluster to apply the log entries to the underlying, local
// version of the KVStore (localStore). If allowGetOnFollower is
// true, the ReplicatedStore will
func NewReplicatedStore(allowGetOnFollower bool, store *LocalStore, cluster *cluster.Cluster) *ReplicatedStore {
	return &ReplicatedStore{
		store:              store,
		cluster:            cluster,
		allowGetOnFollower: allowGetOnFollower,
	}
}

// Get retrieves the value corresponding to the given key; since the
// Get command is non-mutating, it needs not go through the Apply()
// rigmarole; it can be served directly from the LocalStore; if the
// cluster allows it, the values can be served by the followers, which
// can speed things up but there is a non-null probability that the
// follower will serve stale data, when the log has not been
// acknowledged yet, ot it has been acknowledged but not applied yet
// to the local store.
func (s *ReplicatedStore) Get(key string) (string, error) {
	if s.cluster.Raft.State() == raft.Leader || s.allowGetOnFollower {
		log.L.Debug("returning local value")
		return s.store.Get(key)
	}
	log.L.Error("invalid state", zap.Bool("leader", s.cluster.Raft.State() == raft.Leader), zap.Bool("allowed on follower", s.allowGetOnFollower), zap.Error(ErrNotLeader))
	return "", ErrNotLeader
}

// Set sets the value for the given key.
func (s *ReplicatedStore) Set(key, value string) error {
	if s.cluster.Raft.State() != raft.Leader {
		log.L.Error("mutating (set) operation not on Raft cluster leader", zap.Error(ErrNotLeader))
		return ErrNotLeader
	}
	// marshal command and send it over to the FSM via Raft
	b, err := json.Marshal(&Command{
		Type:  Set,
		Key:   key,
		Value: value,
	})
	if err != nil {
		log.L.Error("error marshalling to JSON", zap.Error(err))
		return err
	}

	f := s.cluster.Raft.Apply(b, s.cluster.RaftTimeout)
	return f.Error()
}

// Delete deletes the given key.
func (s *ReplicatedStore) Delete(key string) error {
	if s.cluster.Raft.State() != raft.Leader {
		log.L.Error("mutating operation not on Raft cluster leader", zap.Error(ErrNotLeader))
		return ErrNotLeader
	}
	// marshal command and send it over to the FSM via Raft
	b, err := json.Marshal(&Command{
		Type: Delete,
		Key:  key,
	})
	if err != nil {
		log.L.Error("error marshalling to JSON", zap.Error(err))
		return err
	}

	f := s.cluster.Raft.Apply(b, s.cluster.RaftTimeout)
	return f.Error()
}
