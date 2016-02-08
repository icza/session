/*

Session interface and its implementation.

*/

package session

import (
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
	// A session may be removed automatically if it is not access for this duration.
	Timeout() time.Duration

	// RWMutex returns the RW mutex of the session.
	// It is used to synchronize access/modification of the state stored in the session.
	// It can be used if session-level synchronization is required.
	RWMutex() *sync.RWMutex

	// Access registers an access to the session.
	// Users do not need to call this as the session store is responsible for that.
	Access()
}
