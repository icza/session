/*

An in-memory session store implementation.

*/

package session

import (
	"log"
	"sync"
)

// In-memory session store implementation.
type memStore struct {
	sessions map[string]Session
	mux      *sync.RWMutex
}

// NewMemStore returns a new, in-memory session store.
func NewMemStore() Store {
	// TODO session cleaner
	return &memStore{
		sessions: make(map[string]Session),
		mux:      &sync.RWMutex{},
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
