package session

import (
	"testing"
)

func TestCookieManager_GetStore(t *testing.T) {
	store := NewInMemStore()
	manager := NewCookieManager(store)
	if manager.GetStore() != store {
		t.Fail()
	}
}

func TestCookieManager_GetSessionIdCookieName(t *testing.T) {
	manager := NewCookieManager(NewInMemStore())
	name := manager.GetSessionIdCookieName()
	if name != DEFAULT_SESSION_ID_COOKIE_NAME {
		t.Errorf("Default cookie name has not been set (\"%s\" != \"%s\")", name, DEFAULT_SESSION_ID_COOKIE_NAME)
	}

	const COOKIE_NAME = "SessID-Test"

	opts := CookieMngrOptions{
		SessIDCookieName: COOKIE_NAME,
	}

	manager = NewCookieManagerOptions(manager.GetStore(), &opts)
	name = manager.GetSessionIdCookieName()
	if name != COOKIE_NAME {
		t.Errorf("Bad cookie name (\"%s\" != \"%s\")", name, COOKIE_NAME)
	}
}
