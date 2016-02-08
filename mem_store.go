/*

An in-memory session store implementation.

*/

package session

// In-memory session store implementation.
type memStore struct {
}

// NewMemStore returns a new, in-memory session store.
func NewMemStore() Store {
	// TODO
	return nil
}

// Get returns the session specified by its id.
func (s *memStore) Get(id string) Session {
	// TODO
	return nil
}

// Add adds a new session to the store.
func (s *memStore) Add(sess Session) {
	// TODO
}

// Remove removes a session from the store.
func (s *memStore) Remove(sess Session) {
	// TODO
}
