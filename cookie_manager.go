/*

A secure, cookie based session Manager implementation.

*/

package session

import (
	"net/http"
	"time"
)

// CookieManager is a secure, cookie based session Manager implementation.
// Only the session ID is transmitted / stored at the clients, and it is managed using cookies.
type CookieManager struct {
	store Store // Backing Store

	sessIDCookieName string // Name of the cookie used for storing the session id
	cookieSecure     bool   // Tells if session ID cookies are to be sent only over HTTPS
	cookieMaxAgeSec  int    // Max age for session ID cookies in seconds
	cookiePath       string // Cookie path to use
}

// CookieMngrOptions defines options that may be passed when creating a new CookieManager.
// All fields are optional; default value will be used for any field that has the zero value.
type CookieMngrOptions struct {
	// Name of the cookie used for storing the session id; default value is "sessid"
	SessIDCookieName string

	// Tells if session ID cookies are allowed to be sent over unsecure HTTP too (else only HTTPS);
	// default value is false (only HTTPS)
	AllowHTTP bool

	// Max age for session ID cookies; default value is 30 days
	CookieMaxAge time.Duration

	// Cookie path to use; default value is the root: "/"
	CookiePath string
}

// Pointer to zero value of CookieMngrOptions to be reused for efficiency.
var zeroCookieMngrOptions = new(CookieMngrOptions)

// NewCookieManager creates a new, cookie based session Manager with default options.
// Default values of options are listed in the CookieMngrOptions type.
func NewCookieManager(store Store) Manager {
	return NewCookieManagerOptions(store, zeroCookieMngrOptions)
}

// NewCookieManagerOptions creates a new, cookie based session Manager with the specified options.
func NewCookieManagerOptions(store Store, o *CookieMngrOptions) Manager {
	m := &CookieManager{
		store:            store,
		cookieSecure:     !o.AllowHTTP,
		sessIDCookieName: o.SessIDCookieName,
		cookiePath:       o.CookiePath,
	}

	if m.sessIDCookieName == "" {
		m.sessIDCookieName = "sessid"
	}
	if o.CookieMaxAge == 0 {
		m.cookieMaxAgeSec = 30 * 24 * 60 * 60 // 30 days max age
	} else {
		m.cookieMaxAgeSec = int(o.CookieMaxAge.Seconds())
	}
	if m.cookiePath == "" {
		m.cookiePath = "/"
	}

	return m
}

// Get is to implement Manager.Get().
func (m *CookieManager) Get(r *http.Request) Session {
	c, err := r.Cookie(m.sessIDCookieName)
	if err != nil {
		return nil
	}

	return m.store.Get(c.Value)
}

// Add is to implement Manager.Add().
func (m *CookieManager) Add(sess Session, w http.ResponseWriter) {
	// HttpOnly: do not allow non-HTTP access to it (like javascript) to prevent stealing it...
	// Secure: only send it over HTTPS
	// MaxAge: to specify the max age of the cookie in seconds, else it's a session cookie and gets deleted after the browser is closed.

	c := http.Cookie{
		Name:     m.sessIDCookieName,
		Value:    sess.ID(),
		Path:     m.cookiePath,
		HttpOnly: true,
		Secure:   m.cookieSecure,
		MaxAge:   m.cookieMaxAgeSec,
	}
	http.SetCookie(w, &c)

	m.store.Add(sess)
}

// Remove is to implement Manager.Remove().
func (m *CookieManager) Remove(sess Session, w http.ResponseWriter) {
	// Set the cookie with empty value and 0 max age
	c := http.Cookie{
		Name:     m.sessIDCookieName,
		Value:    "",
		Path:     m.cookiePath,
		HttpOnly: true,
		Secure:   m.cookieSecure,
		MaxAge:   -1, // MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	}
	http.SetCookie(w, &c)

	m.store.Remove(sess)
}

// Close is to implement Manager.Close().
func (m *CookieManager) Close() {
	m.store.Close()
}
