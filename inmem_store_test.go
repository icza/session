package session

import (
	"testing"
	"time"
)

func TestInMemStore(t *testing.T) {
	mt := myt{t}
	eq, neq := mt.eq, mt.neq

	st := NewInMemStore()

	eq(nil, st.Get("asdf"))

	s := NewSession()
	st.Add(s)
	time.Sleep(10 * time.Millisecond)
	eq(s, st.Get(s.Id()))
	neq(s.Accessed(), s.Created())

	st.Remove(s)
	eq(nil, st.Get(s.Id()))
}
