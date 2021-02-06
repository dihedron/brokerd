package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

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

	pairs := []pair{}

	defer f.rows.Close()
	for f.rows.Next() {
		var key string
		var value string
		err := f.rows.Scan(&key, &value)
		if err != nil {
			log.L.Error("error reading value from database", zap.Error(err))
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	// defer sink.Close()

	// TODO: read all rows, create an array and then dump as JSON (for simplicity).
	// once this works, use protobuf instead.

	err := func() error {
		// Encode data.
		b, err := json.Marshal(f.store)
		if err != nil {
			return err
		}

		// Write data to sink.
		if _, err := sink.Write(b); err != nil {
			return err
		}

		// Close the sink.
		return sink.Close()
	}()
	if err != nil {
		sink.Cancel()
	}

	return err
}

func (f *fsmSnapshot) Release() {}
