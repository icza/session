/*

A global session Manager and delegator functions - for easy to use.

*/

package session

import (
	"net/http"
)

// A global session Manager to which the global functions delegate to.
// You may replace this and keep using the global functions, but if you intend to do so,
// you should close it first (e.g. Global.Close()).
var Global Manager = NewCookieManager(NewMemStore())

// Get delegates to Global.Get(); returns the session specified by the HTTP request.
// nil is returned if the request does not contain a session, or the contained session is not know by this manager.
func Get(r *http.Request) Session {
	return Global.Get(r)
}

// Add delegates to Global.Add90; adds the session to the HTTP response.
// This means to let the client know about the specified session by including the sesison id in the response somehow.
func Add(sess Session, w http.ResponseWriter) {
	Global.Add(sess, w)
}

// Remove delegates to Global.Remove(); removes the session from the HTTP response.
func Remove(sess Session, w http.ResponseWriter) {
	Global.Remove(sess, w)
}

// Close delegates to Global.Close(); closes the session manager, releasing any resources that were allocated.
func Close() {
	Global.Close()
}
