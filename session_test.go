package session

import (
	"encoding/base64"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type myt struct {
	*testing.T
}

// eq checks if exp and got are equal, and if not, reports it as an error.
func (m myt) eq(exp, got interface{}) {
	if exp != got {
		_, file, line, _ := runtime.Caller(1)
		m.T.Errorf("[%s:%d] Expected: %v, got: %v", file, line, exp, got)
	}
}

// neq checks if v1 and v2 are not equal, and if they are, reports it as an error.
func (m myt) neq(v1, v2 interface{}) {
	if v1 == v2 {
		_, file, line, _ := runtime.Caller(1)
		m.T.Errorf("[%s:%d] Expected mismatch: %v, got: %v", file, line, v1, v2)
	}
}

func TestNewSession(t *testing.T) {
	mt := myt{t}
	eq, neq := mt.eq, mt.neq
	ss := []Session{NewSession(), NewSessionOptions(&SessOptions{})}

	for _, s := range ss {
		eq(true, s.New())
		eq(s.Accessed(), s.Created())
		eq(0, len(s.Attrs()))
		neq(nil, s.Mutex())

		time.Sleep(10 * time.Millisecond)
		s.Access()
		neq(s.Accessed(), s.Created())
	}
}

func TestSessionAttrs(t *testing.T) {
	mt := myt{t}
	eq := mt.eq
	s := NewSession()

	eq(nil, s.Attr("a"))
	s.SetAttr("a", 1)
	eq(1, s.Attr("a"))
	eq(1, len(s.Attrs()))

	s.SetAttr("a", nil)
	eq(nil, s.Attr("a"))
	eq(0, len(s.Attrs()))
}

func TestSessOptions(t *testing.T) {
	mt := myt{t}
	eq := mt.eq

	so := &SessOptions{
		Attrs:    map[string]interface{}{"a": 1},
		CAttrs:   map[string]interface{}{"ca": 2},
		IdLength: 9,
		Timeout:  43 * time.Minute,
	}

	s := NewSessionOptions(so)

	eq(true, reflect.DeepEqual(s.Attrs(), so.Attrs))

	for k, v := range so.CAttrs {
		eq(v, s.CAttr(k))
	}

	data, err := base64.URLEncoding.DecodeString(s.Id())
	eq(nil, err)
	eq(so.IdLength, len(data))

	eq(so.Timeout, s.Timeout())
}
