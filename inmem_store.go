/*

An in-memory session store implementation.

*/

package session

import (
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

// In-memory session Store implementation.
type inMemStore struct {
	sessions    map[string]Session     // Map of sessions (mapped from ID)
	mux         *sync.RWMutex          // mutex to synchronize access to sessions
	ticker      *time.Ticker           // Ticker for the session cleaner
	closeTicker chan struct{}          // Channel to signal close for the session cleaner
	logPrintln  func(v ...interface{}) // Function used to log session lifecycle events (e.g. added, removed, timed out).
}

// NoopLogger that may be used as InMemStoreOptions.Logger to disable logging.
var NoopLogger = log.New(ioutil.Discard, "", 0)

// InMemStoreOptions defines options that may be passed when creating a new in-memory Store.
// All fields are optional; default value will be used for any field that has the zero value.
type InMemStoreOptions struct {
	// Session cleaner check interval, default is 10 seconds.
	SessCleanerInterval time.Duration

	// Logger to log session lifecycle events (e.g. added, removed, timed out).
	// Default is to use the global functions of the log package.
	// To disable logging, you may use NoopLogger.
	Logger *log.Logger
}

// Pointer to zero value of InMemStoreOptions to be reused for efficiency.
var zeroInMemStoreOptions = new(InMemStoreOptions)

// NewInMemStore returns a new, in-memory session Store with the default options.
// Default values of options are listed in the InMemStoreOptions type.
// The returned Store has an automatic session cleaner which runs
// in its own goroutine.
func NewInMemStore() Store {
	return NewInMemStoreOptions(zeroInMemStoreOptions)
}

// NewInMemStoreOptions returns a new, in-memory session Store with the specified options.
// The returned Store has an automatic session cleaner which runs
// in its own goroutine.
func NewInMemStoreOptions(o *InMemStoreOptions) Store {
	s := &inMemStore{
		sessions:    make(map[string]Session),
		mux:         &sync.RWMutex{},
		closeTicker: make(chan struct{}),
	}

	output := log.Output
	if o.Logger != nil {
		output = o.Logger.Output
	}
	s.logPrintln = func(v ...interface{}) {
		output(3, fmt.Sprintln(v...))
	}

	interval := o.SessCleanerInterval
	if interval == 0 {
		interval = 10 * time.Second
	}

	go s.sessCleaner(interval)

	return s
}

// sessCleaner periodically checks whether sessions have timed out
// in an endless loop. If a session has timed out, removes it.
// This method is to be started as a new goroutine.
func (s *inMemStore) sessCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-s.closeTicker:
			// We are being shut down...
			ticker.Stop()
			return
		case now := <-ticker.C:
			// Do a sweep.
			// Remove is very rare compared to the number of checks, so:
			// "Quick" check with read-lock to see if there's anything to remove:
			// Note: Session.Access() is called with s.mux, the same mutex we use
			// when looking for timed-out sessions, so we're good.
			needRemove := func() bool {
				s.mux.RLock() // Read lock is enough
				defer s.mux.RUnlock()

				for _, sess := range s.sessions {
					if now.Sub(sess.Accessed()) > sess.Timeout() {
						return true
					}
				}
				return false
			}()
			if !needRemove {
				continue
			}

			// Remove required:
			func() {
				s.mux.Lock() // Read-write lock required
				defer s.mux.Unlock()

				for _, sess := range s.sessions {
					if now.Sub(sess.Accessed()) > sess.Timeout() {
						s.logPrintln("Session timed out:", sess.ID())
						delete(s.sessions, sess.ID())
					}
				}
			}()
		}
	}
}

// Get is to implement Store.Get().
func (s *inMemStore) Get(id string) Session {
	s.mux.RLock()
	defer s.mux.RUnlock()

	sess := s.sessions[id]
	if sess == nil {
		return nil
	}

	sess.Access()
	return sess
}

// Add is to implement Store.Add().
func (s *inMemStore) Add(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.logPrintln("Session added:", sess.ID())
	s.sessions[sess.ID()] = sess
}

// Remove is to implement Store.Remove().
func (s *inMemStore) Remove(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.logPrintln("Session removed:", sess.ID())
	delete(s.sessions, sess.ID())
}

// Close is to implement Store.Close().
func (s *inMemStore) Close() {
	close(s.closeTicker)
}
