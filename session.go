/*

Session interface and its implementation.

*/

package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"sync"
	"time"
)

// Session is the (HTTP) session interface.
// We can use it to store and retrieve constant and variable attributes from it.
type Session interface {
	// Id returns the id of the session.
	Id() string

	// New tells if the session is new.
	// Implementation is based on whether created and access times are equal.
	New() bool

	// CAttr returns the value of an attribute provided at session creation.
	// These attributes cannot be changes during the lifetime of a session,
	// so they can be accessed safely without synchronization. Exampe is storing the
	// authenticated user.
	CAttr(name string) interface{}

	// Attr returns the value of an attribute stored in the session.
	// Safe for concurrent use.
	Attr(name string) interface{}

	// SetAttr sets the value of an attribute stored in the session.
	// Pass the nil value to delete the attribute.
	// Safe for concurrent use.
	SetAttr(name string, value interface{})

	// Attrs returns a copy of all the attribute values stored in the session.
	// Safe for concurrent use.
	Attrs() map[string]interface{}

	// Created returns the session creation time.
	Created() time.Time

	// Accessed returns the time when the session was last accessed.
	Accessed() time.Time

	// Timeout returns the session timeout.
	// A session may be removed automatically if it is not accessed for this duration.
	Timeout() time.Duration

	// Mutex returns the RW mutex of the session.
	// It is used to synchronize access/modification of the state stored in the session.
	// It can be used if session-level synchronization is required.
	// Important! If Session values are marshalled / unmarshalled
	// (e.g. multi server instance environment such as Google AppEngine),
	// this mutex may be different for each Session value and thus
	// it can only be used to session-value level synchronization!
	Mutex() *sync.RWMutex

	// Access registers an access to the session,
	// updates its last accessed time to the current time.
	// Users do not need to call this as the session store is responsible for that.
	Access()
}

// Session implementation.
// Fields are exported so a session may be marshalled / unmarshalled.
type sessionImpl struct {
	Id_       string                 // Id of the session
	Created_  time.Time              // Creation time
	Accessed_ time.Time              // Last accessed time
	CAttrs_   map[string]interface{} // Constant attributes specified at session creation
	Attrs_    map[string]interface{} // Attributes stored in the session
	Timeout_  time.Duration          // Session timeout
	mux       *sync.RWMutex          // RW mutex to synchronize session state access
}

// SessOptions defines options that may be passed when creating a new Session.
// All fields are optional; default value will be used for any field that has the zero value.
type SessOptions struct {
	// Constant attributes of the session. These will available via the Session.CAttr() method, without synchronization.
	// Values from the map will be copied, and will be available via Session.CAttr().
	CAttrs map[string]interface{}

	// Initial, non-constant attributes to be stored in the session.
	// Values from the map will be copied, and will be available via Session.Attr() and Session.Attrs,
	// and may be changed with Session.SetAttr().
	Attrs map[string]interface{}

	// Session timeout, default is 30 minutes
	Timeout time.Duration

	// Byte-length of the information that builds up the session ids.
	// Using Base-64 encoding, id length will be this multiplied by 4/3 chars.
	// Default value is 18 (which means length of ID will be 24 chars).
	IdLength int
}

// NewSession creates a new Session with the default options.
// Default values of options are listed in the SessOptions type.
func NewSession() Session {
	return NewSessionOptions(&SessOptions{})
}

// NewSessionOptions creates a new Session with the specified options.
func NewSessionOptions(o *SessOptions) Session {
	now := time.Now()
	idLength := o.IdLength
	if idLength <= 0 {
		idLength = 18
	}
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	sess := sessionImpl{
		Id_:       genId(idLength),
		Created_:  now,
		Accessed_: now,
		Attrs_:    make(map[string]interface{}),
		Timeout_:  timeout,
		mux:       &sync.RWMutex{},
	}

	if len(o.CAttrs) > 0 {
		sess.CAttrs_ = make(map[string]interface{}, len(o.CAttrs))
		for k, v := range o.CAttrs {
			sess.CAttrs_[k] = v
		}
	}

	for k, v := range o.Attrs {
		sess.Attrs_[k] = v
	}

	return &sess
}

// genId generates a secure, random session id using the crypto/rand package.
func genId(length int) string {
	r := make([]byte, length)
	io.ReadFull(rand.Reader, r)
	return base64.URLEncoding.EncodeToString(r)
}

// Id is to implement Session.Id().
func (s *sessionImpl) Id() string {
	return s.Id_
}

// New is to implement Session.New().
func (s *sessionImpl) New() bool {
	return s.Created_ == s.Accessed_
}

// CAttr is to implement Session.CAttr().
func (s *sessionImpl) CAttr(name string) interface{} {
	return s.CAttrs_[name]
}

// Attr is to implement Session.Attr().
func (s *sessionImpl) Attr(name string) interface{} {
	s.mux.RLock()
	defer s.mux.RUnlock()

	return s.Attrs_[name]
}

// SetAttr is to implement Session.SetAttr().
func (s *sessionImpl) SetAttr(name string, value interface{}) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if value == nil {
		delete(s.Attrs_, name)
	} else {
		s.Attrs_[name] = value
	}
}

// Attrs is to implement Session.Attrs().
func (s *sessionImpl) Attrs() map[string]interface{} {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m := make(map[string]interface{}, len(s.Attrs_))
	for k, v := range s.Attrs_ {
		m[k] = v
	}
	return m
}

// Created is to implement Session.Created().
func (s *sessionImpl) Created() time.Time {
	return s.Created_
}

// Accessed is to implement Session.Accessed().
func (s *sessionImpl) Accessed() time.Time {
	return s.Accessed_
}

// Timeout is to implement Session.Timeout().
func (s *sessionImpl) Timeout() time.Duration {
	return s.Timeout_
}

// Mutex is to implement Session.Mutex().
func (s *sessionImpl) Mutex() *sync.RWMutex {
	return s.mux
}

// Access is to implement Session.Access().
func (s *sessionImpl) Access() {
	s.Accessed_ = time.Now()
}
