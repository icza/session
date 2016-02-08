/*

Session Manager interface.

*/

package session

import (
	"net/http"
)

// Manager is a session manager interface.
// A session manager is responsible to acquire a Session from an (incoming) HTTP request,
// and to add a Session to an HTTP response to let the client know about the session.
// A Manager has a backing Store which is responsible to manage Session values at server side.
type Manager interface {
	// Get returns the session specified by the HTTP request.
	// nil is returned if the request does not contain a session, or the contained session is not know by this manager.
	Get(r *http.Request) Session

	// Add adds the session to the HTTP response.
	// This means to let the client know about the specified session by including the sesison id in the response somehow.
	Add(sess Session, w http.ResponseWriter)

	// Remove removes the session from the HTTP response.
	Remove(sess Session, w http.ResponseWriter)

	// Close closes the session manager, releasing any resources that were allocated.
	Close()
}
