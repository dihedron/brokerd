package store4

import (
	"database/sql"
	"encoding/json"

	"github.com/dihedron/brokerd/log"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

// SQLiteFSMSnapshot is a transient object, capable of generating
// a snaphot of the SQLite DB contents at the moment it was created.
type SQLiteFSMSnapshot struct {
	db   *sql.DB
	tx   *sql.Tx
	rows *sql.Rows
}

type pair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Persist writes the SQLiteFSMSnapshot contents to the Raft-provided
// sink.
func (s *SQLiteFSMSnapshot) Persist(sink raft.SnapshotSink) error {
	log.L.Debug("persisting snapshot...")

	// run the transaction inside a nested function, so
	// if anything goes wrong we can capture the error and
	// cancel the snapshot; the nested function must either
	// commit the transaction, write to the sink and close it,
	// or return an error so the transaction is rolled back
	// and the sink can be closed.
	err := func() error {
		pairs := []pair{}
		// loop over the rows and scan them one by one, adding them
		// to the slice; then marshal the slice to JSON and write it
		// out to the sink.
		defer s.rows.Close()
		for s.rows.Next() {
			var key string
			var value string
			if err := s.rows.Scan(&key, &value); err != nil {
				log.L.Error("error reading value from database", zap.Error(err))
				return err
			}
			pairs = append(pairs, pair{Key: key, Value: value})
			log.L.Debug("adding pair to snapshot", zap.String("key", key), zap.String("value", value))
		}
		if err := s.rows.Err(); err != nil {
			log.L.Error("error reading rows", zap.Error(err))
			return err
		}
		// encode data as JSON
		data, err := json.MarshalIndent(pairs, "", "  ")
		if err != nil {
			log.L.Error("error marshalling snapshot to JSON", zap.Error(err))
			return err
		}
		// write data to sink
		if _, err := sink.Write(data); err != nil {
			log.L.Error("error writing snapshot to sink", zap.Error(err))
			return err
		}
		// commit the transaction
		s.tx.Commit()
		// close the sink
		return sink.Close()
	}()
	if err != nil {
		log.L.Error("an error occurred, snapshot cancelled", zap.Error(err))
		s.tx.Rollback()
		sink.Cancel()
	}
	return err
}

// Release is called when a snapshot can be dismissed,
// so any resources and locks can be removed.
func (s *SQLiteFSMSnapshot) Release() {
	log.L.Debug("releasing snapshot")
}
