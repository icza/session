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

// Session is the HTTP session interface.
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

	// RWMutex returns the RW mutex of the session.
	// It is used to synchronize access/modification of the state stored in the session.
	// It can be used if session-level synchronization is required.
	RWMutex() *sync.RWMutex

	// Access registers an access to the session.
	// Users do not need to call this as the session store is responsible for that.
	Access()
}

// Session implementation.
type sessionImpl struct {
	id       string                 // Id of the session
	created  time.Time              // Creation time
	accessed time.Time              // Last accessed time
	cattrs   map[string]interface{} // Constant attributes specified at session creation
	attrs    map[string]interface{} // Attributes stored in the session
	timeout  time.Duration          // Session timeout
	rwMutex  *sync.RWMutex          // RW mutex to synchronize session state access
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
	// Using Base-64 encoding id string will be up to this multiplied by 4/3 chars.
	// Default value is 18.
	IdLength int
}

// NewSession creates a new Session with the default options.
// Default options are listed in the SessOptions type.
func NewSession() Session {
	return NewSessionOptions(&SessOptions{})
}

// NewSessionOptions creates a new Session with the specified options.
func NewSessionOptions(o *SessOptions) Session {
	now := time.Now()
	idLength := o.IdLength
	if idLength == 0 {
		idLength = 18
	}
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	sess := sessionImpl{
		id:       genId(idLength),
		created:  now,
		accessed: now,
		attrs:    make(map[string]interface{}),
		timeout:  timeout,
		rwMutex:  &sync.RWMutex{},
	}

	if len(o.CAttrs) > 0 {
		sess.cattrs = make(map[string]interface{}, len(o.CAttrs))
		for k, v := range o.CAttrs {
			sess.cattrs[k] = v
		}
	}

	for k, v := range o.Attrs {
		sess.attrs[k] = v
	}

	return &sess
}

// genId generates a secure, random session id using the crypto/rand package.
func genId(length int) string {
	r := make([]byte, length)
	io.ReadFull(rand.Reader, r)
	return base64.RawURLEncoding.EncodeToString(r)
}

func (s *sessionImpl) Id() string {
	return s.id
}

func (s *sessionImpl) New() bool {
	return s.created == s.accessed
}

func (s *sessionImpl) CAttr(name string) interface{} {
	return s.cattrs[name]
}

func (s *sessionImpl) Attr(name string) interface{} {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	return s.attrs[name]
}

func (s *sessionImpl) SetAttr(name string, value interface{}) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	if value == nil {
		delete(s.attrs, name)
	} else {
		s.attrs[name] = value
	}
}

func (s *sessionImpl) Attrs() map[string]interface{} {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	m := make(map[string]interface{}, len(s.attrs))
	for k, v := range s.attrs {
		m[k] = v
	}
	return m
}

func (s *sessionImpl) Created() time.Time {
	return s.created
}

func (s *sessionImpl) Accessed() time.Time {
	return s.accessed
}

func (s *sessionImpl) Timeout() time.Duration {
	return s.timeout
}

func (s *sessionImpl) RWMutex() *sync.RWMutex {
	return s.rwMutex
}

func (s *sessionImpl) Access() {
	s.accessed = time.Now()
}
