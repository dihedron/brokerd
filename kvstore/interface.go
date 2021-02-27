package kvstore

// KVStore is the common interface to all key/value stores.
type KVStore interface {
	// Get retrieves a value from the store, given its key.
	Get(key string) (string, error)
	// Set sets a value into the store, creating it if non existing.
	Set(key string, value string) error
	// Delete removes a key/value pair from the store.
	Delete(key string) error
}
