package session

import (
	"testing"
	"time"

	"github.com/icza/mighty"
)

func TestCookieManager(t *testing.T) {
	eq := mighty.Eq(t)

	o := &CookieMngrOptions{
		SessIDCookieName: "test",
		AllowHTTP:        true,
		CookieMaxAge:     time.Second * 1234,
		CookiePath:       "/testpath",
	}
	mgr := NewCookieManagerOptions(nil, o)

	cmgr := mgr.(*CookieManager)

	eq(o.SessIDCookieName, cmgr.SessIDCookieName())
	eq(!o.AllowHTTP, cmgr.CookieSecure())
	eq(int(o.CookieMaxAge/time.Second), cmgr.CookieMaxAgeSec())
	eq(o.CookiePath, cmgr.CookiePath())
}
