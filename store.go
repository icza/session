/*

Session store interface.

*/

package session

// Store is a session store interface.
type Store interface {
	// Get returns the session specified by its id.
	// The returned session will have an updated access time (set to the current time).
	Get(id string) Session

	// Add adds a new session to the store.
	Add(sess Session)

	// Remove removes a session from the store.
	Remove(sess Session)

	// Close closes the session store, releasing any resources that were allocated.
	Close()
}
