package session

import (
	"encoding/base64"
	"reflect"
	"testing"
	"time"

	"github.com/icza/mighty"
)

func TestNewSession(t *testing.T) {
	eq, neq := mighty.EqNeq(t)
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
	eq := mighty.Eq(t)
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
	eq := mighty.Eq(t)

	so := &SessOptions{
		Attrs:    map[string]interface{}{"a": 1},
		CAttrs:   map[string]interface{}{"ca": 2},
		IDLength: 9,
		Timeout:  43 * time.Minute,
	}

	s := NewSessionOptions(so)

	eq(true, reflect.DeepEqual(s.Attrs(), so.Attrs))

	for k, v := range so.CAttrs {
		eq(v, s.CAttr(k))
	}

	data, err := base64.URLEncoding.DecodeString(s.ID())
	eq(nil, err)
	eq(so.IDLength, len(data))

	eq(so.Timeout, s.Timeout())
}
