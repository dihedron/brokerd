package cluster

import (
	"fmt"
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
type Cluster struct {
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
	// NodeID is the unique ID of the server in the cluster.
	NodeID string
	// Raft is the underlying Raft consensus cluster.
	Raft *raft.Raft
	// Transport is the underlying transport layer for Raft.
	Transport *raft.NetworkTransport
	// Snapshots is the underlying snapshots store.
	Snapshots *raft.FileSnapshotStore
}

// New creates a new Cluster and associates it with the given finite
// state machine (FSM), with the given cluster options.
func New(nodeID string, fsm raft.FSM, options ...Option) (*Cluster, error) {
	// setup with defaults
	c := &Cluster{
		NodeID:                  nodeID,
		RaftDirectory:           "raft",
		RaftBindAddress:         "127.0.0.1:12000",
		RaftRetainSnapshotCount: DefaultRetainSnapshotCount,
	}
	// apply functional options to override
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
	c.Transport = transport

	// create the snapshot store; this allows the Raft to truncate the log
	snapshots, err := raft.NewFileSnapshotStore(c.RaftDirectory, DefaultRetainSnapshotCount, os.Stderr)
	if err != nil {
		log.L.Error("error creating file snaphost store", zap.String("directory", c.RaftDirectory), zap.Error(err))
		return nil, fmt.Errorf("file snapshot store: %s", err)
	}
	c.Snapshots = snapshots

	// create the underlying BoltDB store, used as both log store and stable store
	// var logStore raft.LogStore
	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(c.RaftDirectory, "raft.db"))
	if err != nil {
		return nil, fmt.Errorf("new bolt store: %s", err)
	}

	// instantiate the Raft systems
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	r, err := raft.NewRaft(config, fsm, boltDB, boltDB, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("new raft: %s", err)
	}
	c.Raft = r
	return c, nil
}

// Node represents a node in the Cluster.
type Node struct {
	ID      string
	Address string
}

// Bootstrap bootstraps the Cluster with the set of nodes; if none is
// provided, the cluster is bootstrapped with this single node.
func (c *Cluster) Bootstrap(nodes ...Node) error {
	configuration := raft.Configuration{
		Servers: []raft.Server{},
	}
	if len(nodes) == 0 {
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      raft.ServerID(c.NodeID),
			Address: c.Transport.LocalAddr(),
		})
	} else {
		for _, node := range nodes {
			configuration.Servers = append(configuration.Servers, raft.Server{
				ID:      raft.ServerID(node.ID),
				Address: raft.ServerAddress(node.Address),
			})
		}
	}

	if f := c.Raft.BootstrapCluster(configuration); f.Error() != nil {
		log.L.Error("error bootstrapping cluster", zap.Error(f.Error()))
		return f.Error()
	}
	log.L.Info("cluster bootstrapped successfully", zap.String("master node ID", c.NodeID))
	return nil
}

// Join joins a node, identified by nodeID and located at address, to
// this cluster. The node must be ready to respond to Raft communications
// at that address.
func (c *Cluster) Join(nodeID string, address string) error {
	log.L.Info("received join request for remote node", zap.String("nodeID", nodeID), zap.String("address", address))

	configFuture := c.Raft.GetConfiguration()
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

			future := c.Raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, address, err)
			}
		}
	}

	if f := c.Raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(address), 0, 0); f.Error() != nil {
		log.L.Error("error adding voter to cluster", zap.Error(f.Error()), zap.String("node ID", nodeID), zap.String("address", address))
		return f.Error()
	}
	log.L.Info("node joined successfully", zap.String("node ID", nodeID), zap.String("address", address))
	return nil
}
