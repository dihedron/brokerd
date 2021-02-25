package store

import "time"

const (
	// DefaultRetainSnapshotCount is the default number of snaphots to keep.
	DefaultRetainSnapshotCount = 2
	// DefaultRaftTimeout is the default timeout of the Raft cluster.
	DefaultRaftTimeout = 10 * time.Second
)

// Options contains all options which will be applied when instantiating a Store.
type Options struct {
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
}

// Option represents the optional function.
type Option func(opts *Options)

// WithOptions accepts the whole options config.
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithRaftDirectory sets up the directory where Raft protocol will
// be kept.
func WithRaftDirectory(value string) Option {
	return func(opts *Options) {
		opts.RaftDirectory = value
	}
}

// WithRaftBindAddress sets up the network address on which the Raft
// protocol endpoint will be made available.
func WithRaftBindAddress(value string) Option {
	return func(opts *Options) {
		opts.RaftBindAddress = value
	}
}

// WithRaftRetainSnapshotCount sets up the number of Raft snapshots
// to keep.
func WithRaftRetainSnapshotCount(value int) Option {
	return func(opts *Options) {
		opts.RaftRetainSnapshotCount = value
	}
}

// WithRaftTimeout sets up the timeout of the Raft cluster.
func WithRaftTimeout(value time.Duration) Option {
	return func(opts *Options) {
		opts.RaftTimeout = value
	}
}

// loadOptions applies the functional options one at a time, progressively
// populating an Options struct, which is eventually returned.
func loadOptions(options ...Option) *Options {
	// load defaults
	opts := &Options{
		RaftDirectory:           "raft",
		RaftBindAddress:         "127.0.0.1:12000",
		RaftRetainSnapshotCount: DefaultRetainSnapshotCount,
	}
	// apply functional options
	for _, option := range options {
		option(opts)
	}
	return opts
}
