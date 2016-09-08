package session

import (
	"testing"
	"time"
)

func TestInMemStore(t *testing.T) {
	mt := myt{t}
	eq, neq := mt.eq, mt.neq

	st := NewInMemStore()
	defer st.Close()

	eq(nil, st.Get("asdf"))

	s := NewSession()
	st.Add(s)
	time.Sleep(10 * time.Millisecond)
	eq(s, st.Get(s.Id()))
	neq(s.Accessed(), s.Created())

	st.Remove(s)
	eq(nil, st.Get(s.Id()))
}

func TestInMemStoreSessCleaner(t *testing.T) {
	mt := myt{t}
	eq := mt.eq

	st := NewInMemStoreOptions(&InMemStoreOptions{SessCleanerInterval: 10 * time.Millisecond})
	defer st.Close()

	s := NewSessionOptions(&SessOptions{Timeout: 50 * time.Millisecond})
	st.Add(s)
	eq(s, st.Get(s.Id()))

	time.Sleep(30 * time.Millisecond)
	eq(s, st.Get(s.Id()))

	time.Sleep(80 * time.Millisecond)
	eq(nil, st.Get(s.Id()))
}
