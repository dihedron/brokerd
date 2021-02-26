package store2

/*

// ReplicatedSQLiteStore provides Raft-base replication
// of the SQLiteStore across the cluster.
type ReplicatedSQLiteStore struct {
	// store is the actual SQLiteStore.
	store *SQLiteStore
	// cluster is the Raft consensus mechanism.
	cluster *cluster.Cluster
}

// NewReplicatedSQLiteStore creates and initialises a new SQLite-based store.
func NewReplicatedSQLiteStore(store *SQLiteStore, cluster *cluster.Cluster) (*ReplicatedSQLiteStore, error) {
	if store == nil {
		log.L.Error("SQLite store must not be null")
		return nil, fmt.Errorf("invalid SQLite store reference")
	}
	if cluster == nil {
		log.L.Error("Raft cluster must not be null")
		return nil, fmt.Errorf("invalid Raft cluster store reference")
	}
	return &ReplicatedSQLiteStore{
		store:   store,
		cluster: cluster,
	}, nil
}

// // Get returns the value for the given key.
// func (s *ReplicatedSQLiteStore) Get(key string) (string, error) {
// 	tx, err := s.store.db.BeginTx(context.Background(), &sql.TxOptions{
// 		Isolation: sql.LevelDefault,
// 		ReadOnly:  true,
// 	})
// 	if err != nil {
// 		log.L.Error("error opening read-only transaction", zap.Error(err))
// 		tx.Rollback()
// 		return "", err
// 	}
// 	value := ""
// 	if err := tx.QueryRow("SELECT value FROM pairs WHERE key=?", key).Scan(&value); err != nil {
// 		log.L.Error("error querying row", zap.String("key", key), zap.Error(err))
// 		tx.Rollback()
// 		return "", err
// 	}
// 	tx.Commit()
// 	log.L.Debug("returning value", zap.String("key", key), zap.String("value", value))
// 	return value, nil
// }

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
*/
