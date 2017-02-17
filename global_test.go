package session

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/icza/mighty"
)

func globalHandler(w http.ResponseWriter, r *http.Request) {
	if sess := Get(r); sess == nil {
		sess = NewSession()
		sess.SetAttr("counter", 1)
		Add(sess, w)
		w.Header().Set("test", "0")
	} else {
		if sess.Attr("counter") == 1 {
			sess.SetAttr("counter", 2)
			w.Header().Set("test", "1")
		} else {
			Remove(sess, w)
			w.Header().Set("test", "2")
		}
	}
}

func TestGlobal(t *testing.T) {
	eq := mighty.Eq(t)

	Global.Close()
	Global = NewCookieManagerOptions(NewInMemStore(),
		&CookieMngrOptions{AllowHTTP: true, CookieMaxAge: time.Hour})
	defer Close()

	server := httptest.NewServer(http.HandlerFunc(globalHandler))
	defer server.Close()

	jar, err := cookiejar.New(nil)
	eq(nil, err)

	client := &http.Client{Jar: jar}

	// 3 iterations: Create, Change, Remove session
	// And a 4th: it should be Create again due to Remove
	for i := 0; i < 4; i++ {
		resp, err := client.Get(server.URL)
		eq(nil, err)
		eq(strconv.Itoa(i%3), resp.Header.Get("test"))
		resp.Body.Close()
	}
}
