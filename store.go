/*

Session store interface.

*/

package session

// Store is a session store interface.
type Store interface {
	// Get returns the session specified by its id.
	Get(id string) Session

	// Add adds a new session to the store.
	Add(sess Session)

	// Remove removes a session from the store.
	Remove(sess Session)
}
