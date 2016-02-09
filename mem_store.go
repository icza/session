/*

An in-memory session store implementation.

*/

package session

import (
	"log"
	"sync"
	"time"
)

// In-memory session Store implementation.
type memStore struct {
	sessions    map[string]Session // Map of sessions (mapped from ID)
	mux         *sync.RWMutex      // mutex to synchronize access to sessions
	ticker      *time.Ticker       // Ticker for the session cleaner
	closeTicker chan struct{}      // Channel to signal close for the session cleaner
}

// NewMemStore returns a new, in-memory session Store.
// The returned Store has an automatic session cleaner which runs
// in its own goroutine.
func NewMemStore() Store {
	s := &memStore{
		sessions:    make(map[string]Session),
		mux:         &sync.RWMutex{},
		closeTicker: make(chan struct{}),
	}

	go s.sessCleaner()

	return s
}

// sessCleaner periodically checks whether sessions have timed out
// in an endless loop. If a session has timed out, removes it.
// This method is to be started as a new goroutine.
func (s *memStore) sessCleaner() {
	ticker := time.NewTicker(10 * time.Second)

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
						log.Println("Session timed out:", sess.Id())
						delete(s.sessions, sess.Id())
					}
				}
			}()
		}
	}
}

func (s *memStore) Get(id string) Session {
	s.mux.RLock()
	defer s.mux.RUnlock()

	sess := s.sessions[id]
	if sess == nil {
		return nil
	}

	sess.Access()
	return sess
}

func (s *memStore) Add(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	log.Println("Session added:", sess.Id())
	s.sessions[sess.Id()] = sess
}

func (s *memStore) Remove(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	log.Println("Session removed:", sess.Id())
	delete(s.sessions, sess.Id())
}

func (s *memStore) Close() {
	close(s.closeTicker)
}
