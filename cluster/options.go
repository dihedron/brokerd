package cluster

import "time"

const (
	// DefaultRetainSnapshotCount is the default number of snaphots to keep.
	DefaultRetainSnapshotCount = 2
	// DefaultRaftTimeout is the default timeout of the Raft cluster.
	DefaultRaftTimeout = 10 * time.Second
)

// Option represents the optional function.
type Option func(cluster *Cluster)

// WithRaftDirectory sets up the directory where Raft protocol will
// be kept.
func WithRaftDirectory(value string) Option {
	return func(cluster *Cluster) {
		cluster.RaftDirectory = value
	}
}

// WithRaftBindAddress sets up the network address on which the Raft
// protocol endpoint will be made available.
func WithRaftBindAddress(value string) Option {
	return func(cluster *Cluster) {
		cluster.RaftBindAddress = value
	}
}

// WithRaftRetainSnapshotCount sets up the number of Raft snapshots
// to keep.
func WithRaftRetainSnapshotCount(value int) Option {
	return func(cluster *Cluster) {
		cluster.RaftRetainSnapshotCount = value
	}
}

// WithRaftTimeout sets up the timeout of the Raft cluster.
func WithRaftTimeout(value time.Duration) Option {
	return func(cluster *Cluster) {
		cluster.RaftTimeout = value
	}
}
