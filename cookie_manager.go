/*

A secure, cookie based session Manager implementation.

*/

package session

import (
	"net/http"
	"time"
)

// A secure, cookie based session Manager implementation.
// Only the session ID is transmitted / stored at the clients, and it is managed using cookies.
type CookieManager struct {
	store Store // Backing Store
}

// CookieMngrOptions defines options that may be passed when creating a new CookieManager.
// All fields are optional; default value will be used for any field that has the zero value.
type CookieMngrOptions struct {
	SessIdCookieName string        // Name of the cookie used for storing the session id; default value is "sessid"
	CookieHttpsOnly  *bool         // Tells if session ID cookies are to be sent only over HTTPS; default value is true
	CookieMaxAge     time.Duration // Max age for session ID cookies; default value is 30 days
	CookiePath       string        // Cookie path to use; default value is the root: "/"
}

// TODO NewCookieManagerOptions
// func NewCookieManager(store Store) Manager {}

// NewCookieManager returns a new, cookie based session Manager.
func NewCookieManager(store Store) Manager {
	m := &CookieManager{
		store: store,
	}

	return m
}

// Name of the cookie used for storing the session id
var SessIdCookieName = "sessid"

func (m *CookieManager) Get(r *http.Request) Session {
	c, err := r.Cookie(SessIdCookieName)
	if err != nil {
		return nil
	}

	return m.store.Get(c.Value)
}

func (m *CookieManager) Add(sess Session, w http.ResponseWriter) {
	// HttpOnly: do not allow non-HTTP access to it (like javascript) to prevent stealing it...
	// Secure: only send it over HTTPS
	// MaxAge: to specify the max age of the cookie in seconds, else it's a session cookie and gets deleted after the browser is closed.

	c := http.Cookie{
		Name:     SessIdCookieName,
		Value:    sess.Id(),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   30 * 24 * 60 * 60, // 30 days max age
	}
	http.SetCookie(w, &c)

	m.store.Add(sess)
}

func (m *CookieManager) Remove(sess Session, w http.ResponseWriter) {
	// Set the cookie with empty value and 0 max age
	c := http.Cookie{
		Name:     SessIdCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1, // MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	}
	http.SetCookie(w, &c)

	m.store.Remove(sess)
}

func (m *CookieManager) Close() {
	m.store.Close()
}
