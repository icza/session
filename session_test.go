package session

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ss := []Session{NewSession(), NewSessionOptions(&SessOptions{})}

	for _, s := range ss {
		si := s.(*sessionImpl)
		if !si.New() {
			t.Errorf("Session should be new!")
		}
		if si.Created() != si.Accessed() {
			t.Errorf("Created should be equal to Accessed for new sessions!")
		}
		if as := si.Attrs(); len(as) != 0 {
			t.Errorf("New session should have no attrs: %v", as)
		}

		time.Sleep(10 * time.Millisecond)
		si.Access()
		if si.Created() == si.Accessed() {
			t.Errorf("Created should not be equal to Accessed for non-new sessions!")
		}
	}
}

func TestAttrs(t *testing.T) {
	s := NewSession()

	if v := s.Attr("a"); v != nil {
		t.Errorf("Expected: %v, got: %v", nil, v)
	}
	s.SetAttr("a", 1)
	if v := s.Attr("a"); v != 1 {
		t.Errorf("Expected: %v, got: %v", 1, v)
	}
	if v := len(s.Attrs()); v != 1 {
		t.Errorf("Expected: %v, got: %v", 1, v)
	}

	s.SetAttr("a", nil)
	if v := s.Attr("a"); v != nil {
		t.Errorf("Expected: %v, got: %v", nil, v)
	}
	if v := len(s.Attrs()); v != 0 {
		t.Errorf("Expected: %v, got: %v", 1, v)
	}
}
