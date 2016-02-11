# Session

The [Go](https://golang.org/) standard library includes a nice [http server](https://golang.org/pkg/net/http/), but unfortunately it lacks a very basic and important feature: _HTTP session management_.

This package provides an easy-to-use, extensible and secure session implementation and management. Package documentation can be found and godoc.org:

https://godoc.org/github.com/icza/session

This is "just" an HTTP session implementation and management, you can use it as-is, or with any existing Go web toolkits and frameworks.

## Overview

There are 3 key _players_ in the package:

- **`Session`** is the (HTTP) session interface. We can use it to store and retrieve constant and variable attributes from it.
- **`Store`** is a session store interface which is responsible to store sessions and make them retrievable by their IDs at the server side.
- **`Manager`** is a session manager interface which is responsible to acquire a `Session` from an (incoming) HTTP request, and to add a `Session` to an HTTP response to let the client know about the session. A `Manager` has a backing `Store` which is responsible to manage `Session` values at server side.

_Players_ of this package are represented by interfaces, and various implementations are provided for all these players.
You are not bound by the provided implementations, feel free to provide your own implementations for any of the players.

## Usage

Usage can't be simpler than this. To get the current session associated with the [http.Request](https://golang.org/pkg/net/http/#Request):

    sess := session.Get(r)
    if sess == nil {
        // No session (yet)
    } else {
        // We have a session, use it
    }

To create a new session (e.g. on a successful login) and add it to an [http.ResponseWriter](https://golang.org/pkg/net/http/#ResponseWriter) (to let the client know about the session):

    sess := session.NewSession()
    session.Add(sess, w)

Let's see a more advanced session creation: let's provide a constant attribute (for the lifetime of the session) and an initial, variable attribute:

    sess := session.NewSessionOptions(&session.SessOptions{
        CAttrs: map[string]interface{}{"UserName": userName},
        Attrs:  map[string]interface{}{"Count": 1},
    })

And to access these attributes and change value of `"Count"`:

    userName := sess.CAttr("UserName")
    count := sess.Attr("Count").(int) // Type assertion, you might wanna check if it succeeds
    sess.SetAttr("Count", count+1)    // Increment count

(Of course variable attributes can be added later on too with `Session.SetAttr()`, not just at session creation.)

To remove a session (e.g. on logout):

    session.Remove(sess, w)

Check out the [session demo application](https://github.com/icza/session/blob/master/session_demo/session_demo.go) which shows all these in action.

## Google AppEngine support

The package provides support for Google AppEngine (GAE) platform.

The documentation doesn't include it (due to the `+build appengine` build constraint), but here it is: [gae_memcache_store.go](https://github.com/icza/session/blob/master/gae_memcache_store.go)

We can use `NewMemcacheStore()` and `NewMemcacheStoreOptions()` functions to create a session Store implementation
which stores sessions in GAE's Memcache. Important to note that since accessing the Memcache relies on
Appengine Context which is bound to an `http.Request`, the returned Store can only be used for the lifetime of a request!
Note that the Store will automatically "flush" sessions accessed from it when the Store is closed,
so it is very important to close the Store at the end of your request; this is usually done by closing
the session manager to which you passed the store (preferably with the defer statement).

So in each request handling we have to create a new session manager using a new Store, and we can use the session manager
to do session-related tasks, something like this:

    ctx := appengine.NewContext(r)
    sessmgr := session.NewCookieManager(session.NewMemcacheStore(ctx))
    defer sessmgr.Close() // Note the Close(): it will ensure changes made to the session are auto-saved in Memcache.
    
    sess := sessmgr.Get(r) // Get current session
    if sess != nil {
        // Session exists, do something with it.
        ctx.Infof("Count: %v", sess.Attr("Count"))
    } else {
        // No session yet, let's create one and add it:
        sess = session.NewSession()
        sess.SetAttr("Count", 1)
        sessmgr.Add(sess, w)
    }

Check out the [GAE session demo application](https://github.com/icza/session/blob/master/gae_session_demo/gae_session_demo.go) which shows how to use this in action.
