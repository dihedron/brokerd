package store

import (
	"database/sql"
	"encoding/json"

	"github.com/dihedron/brokerd/log"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

type fsmSnapshot struct {
	db   *sql.DB
	tx   *sql.Tx
	rows *sql.Rows
	// store map[string]string
}

type pair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
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
		defer f.rows.Close()
		for f.rows.Next() {
			var key string
			var value string
			if err := f.rows.Scan(&key, &value); err != nil {
				log.L.Error("error reading value from database", zap.Error(err))
				return err
			}
			pairs = append(pairs, pair{Key: key, Value: value})
			log.L.Debug("adding pair to snapshot", zap.String("key", key), zap.String("value", value))
		}
		if err := f.rows.Err(); err != nil {
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
		f.tx.Commit()
		// close the sink
		return sink.Close()
	}()
	if err != nil {
		log.L.Error("an error occurred, snapshot cancelled", zap.Error(err))
		f.tx.Rollback()
		sink.Cancel()
	}

	return err
}

func (f *fsmSnapshot) Release() {}
