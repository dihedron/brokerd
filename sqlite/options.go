package sqlite

const (
	// DefaultStoreDirectory is the default name of the directory
	// where the SQLite data will be kept.
	DefaultStoreDirectory string = "sqlite"
	// DefaultStoreFileName is the default name of the SQLite3 data file.
	DefaultStoreFileName string = "sqlite3.db"
)

// Option represents the optional function.
type Option func(store *Store)

// WithStoreDirectory sets up the directory where the SQLite data will
// be kept.
func WithStoreDirectory(value string) Option {
	return func(store *Store) {
		store.DataDirectory = value
	}
}

// WithStoreFileName sets up the name of the SQLite3 data file.
func WithStoreFileName(value string) Option {
	return func(store *Store) {
		store.DataFileName = value
	}
}
