/*

A global session Manager and delegator functions - for easy to use.

*/

package session

import (
	"net/http"
)

// Global is the default session Manager to which the top-level functions such as Get, Add, Remove and Close
// are wrappers of Manager.
// You may replace this and keep using the top-level functions, but if you intend to do so,
// you should close it first with Global.Close().
var Global = NewCookieManager(NewInMemStore())

// Get delegates to Global.Get(); returns the session specified by the HTTP request.
// nil is returned if the request does not contain a session, or the contained session is not know by this manager.
func Get(r *http.Request) Session {
	return Global.Get(r)
}

// Add delegates to Global.Add(); adds the session to the HTTP response.
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
