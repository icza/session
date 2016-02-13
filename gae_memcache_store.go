// +build appengine

/*

A Google App Engine Memcache session store implementation.

Warning! This imlementation stores sessions only in the Memcache which may lose its content at any time, resulting in losing sessions.
This may or may not be sufficient for your purposes.

Limitations based on GAE Memcache:

- Since session ids are used in the Memcache keys, session ids can't be longer than 250 chars (bytes, but with Base64 charset it's the same).
If you also specify a key prefix (in MemcacheStoreOptions), that also counts into it.

- The size of a Session cannot be larger than 1 MB (marshalled into a byte slice).

Note that the Store will automatically "flush" sessions accessed from it when the Store is closed,
so it is very important to close the Store at the end of your request; this is usually done by closing
the session manager to which you passed the store (preferably with the defer statement).

Check out the GAE session demo application which shows how to use it properly:

https://github.com/icza/session/blob/master/gae_session_demo/session_demo.go

*/

package session

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"sync"
	"time"
)

// A Google App Engine Memcache session store implementation.
type memcacheStore struct {
	ctx appengine.Context // Appengine context used when accessing the Memcache

	keyPrefix string // Prefix to use in front of session ids to construct Memcache key
	retries   int    // Number of retries to perform in case of general Memcache failures

	codec memcache.Codec // Codec used to marshal and unmarshal a Session to a byte slice

	onlyMemcache      bool   // Tells if sessions are not to be saved in Datastore
	syncDatastoreSave bool   // Tells if saving to Datastore should happen synchronously, in the same goroutine
	dsEntityName      string // Name of the datastore entity to use to save sessions

	// Map of sessions (mapped from ID) that were accessed using this store; usually it will only be 1.
	// It is also used as a cache, should the user call Get() with the same id multiple times.
	sessions map[string]Session

	mux *sync.RWMutex // mutex to synchronize access to sessions
}

// MemcacheStoreOptions defines options that may be passed when creating a new Memcache session store.
// All fields are optional; default value will be used for any field that has the zero value.
type MemcacheStoreOptions struct {
	// Prefix to use when storing sessions in the Memcache, cannot contain a null byte
	// and cannot be longer than 250 chars (bytes) when concatenated with the session id; default value is the empty string
	// The Memcache key will be this prefix and the session id concatenated.
	KeyPrefix string

	// Number of retries to perform if Memcache operations fail due to general service error;
	// default value is 3
	Retries int

	// Codec used to marshal and unmarshal a Session to a byte slice;
	// Default value is &memcache.Gob (which uses the gob package).
	Codec *memcache.Codec

	// Tells if sessions are only to be stored in Memcache, and do not store them in Datastore as backup;
	// as Memcache has no guarantees, it may lose content from time to time, but if Datastore is
	// also used, the session will automatically be retrieved from the Datastore if not found in Memcache;
	// default value is false (which means to also save sessions in the Datastore)
	OnlyMemcache bool

	// Tells if saving to Datastore should happen synchronously (in the same goroutine, before returning),
	// if false, session saving to Datastore will happen in the background (in another goroutine)
	// which gives smaller latency (and is enough most of the times as Memcache is always checked first);
	// default value is false which means to save sessions to Datastore in the background and return immedately
	// Not used if OnlyMemcache=true.
	SyncDatastoreSave bool

	// Name of the entity to use for saving sessions;
	// default value is "sess_"
	// Not used if OnlyMemcache=true.
	DSEntityName string
}

// SessEntity models the session entity saved to Datastore.
// The Key is the session id.
type SessEntity struct {
	Expires time.Time `datastore:"exp"`
	Value   []byte    `datastore:"val"`
}

// Pointer to zero value of MemcacheStoreOptions to be reused for efficiency.
var zeroMemcacheStoreOptions = new(MemcacheStoreOptions)

// NewMemcacheStore returns a new, GAE Memcache session Store with default options.
// Default values of options are listed in the MemcacheStoreOptions type.
//
// Important! Since accessing the Memcache relies on Appengine Context
// which is bound to an http.Request, the returned Store can only be used for the lifetime of a request!
func NewMemcacheStore(ctx appengine.Context) Store {
	return NewMemcacheStoreOptions(ctx, zeroMemcacheStoreOptions)
}

// NewMemcacheStoreOptions returns a new, GAE Memcache session Store with the specified options.
//
// Important! Since accessing the Memcache relies on Appengine Context
// which is bound to an http.Request, the returned Store can only be used for the lifetime of a request!
func NewMemcacheStoreOptions(ctx appengine.Context, o *MemcacheStoreOptions) Store {
	s := &memcacheStore{
		ctx:               ctx,
		keyPrefix:         o.KeyPrefix,
		retries:           o.Retries,
		onlyMemcache:      o.OnlyMemcache,
		syncDatastoreSave: o.SyncDatastoreSave,
		dsEntityName:      o.DSEntityName,
		sessions:          make(map[string]Session, 2),
		mux:               &sync.RWMutex{},
	}
	if s.retries <= 0 {
		s.retries = 3
	}
	if o.Codec != nil {
		s.codec = *o.Codec
	} else {
		s.codec = memcache.Gob
	}
	if s.dsEntityName == "" {
		s.dsEntityName = "sess_"
	}
	return s
}

// Get is to implement Store.Get().
// Important! Since sessions are marshalled and stored in the Memcache,
// the mutex of the Session (Session.RWMutex()) will be different for each
// Session value (even though they might have the same session id)!
func (s *memcacheStore) Get(id string) Session {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// First check our "cache"
	if sess := s.sessions[id]; sess != nil {
		return sess
	}

	// Next check in Memcache
	var err error
	var sess *sessionImpl

	for i := 0; i < s.retries; i++ {
		var sess_ sessionImpl
		_, err = s.codec.Get(s.ctx, s.keyPrefix+id, &sess_)
		if err == memcache.ErrCacheMiss {
			break // It's not in the Memcache (e.g. invalid sess id or was removed from Memcache by AppEngine)
		}
		if err == nil {
			sess = &sess_
			break
		}
		// Service error? Retry..
	}

	if sess == nil {
		if err != nil && err != memcache.ErrCacheMiss {
			s.ctx.Errorf("Failed to get session from memcache, id: %s, error: %v", id, err)
		}

		// Ok, we didn't get it from Memcace (either was not there or Memcache service is unavailable).
		// Now it's time to check in the Datastore.
		key := datastore.NewKey(s.ctx, s.dsEntityName, id, 0, nil)
		for i := 0; i < s.retries; i++ {
			e := SessEntity{}
			err = datastore.Get(s.ctx, key, &e)
			if err == datastore.ErrNoSuchEntity {
				return nil // It's not in the Datastore either
			}
			if err != nil {
				// Service error? Retry..
				continue
			}
			if e.Expires.After(time.Now()) {
				// Session expired.
				datastore.Delete(s.ctx, key) // Omitting error check...
				return nil
			}
			var sess_ sessionImpl
			if err = s.codec.Unmarshal(e.Value, &sess_); err != nil {
				break // Invalid data in stored session entity...
			}
			sess = &sess_
			break
		}
	}

	if sess == nil {
		s.ctx.Errorf("Failed to get session from datastore, id: %s, error: %v", id, err)
		return nil
	}

	// Yes! We have it! "Actualize" it.
	sess.Access()
	// Mutex is not marshalled, so create a new one:
	sess.mux = &sync.RWMutex{}
	s.sessions[id] = sess
	return sess
}

// Add is to implement Store.Add().
func (s *memcacheStore) Add(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.setMemcacheSession(sess) {
		s.ctx.Infof("Session added: %s", sess.Id())
		s.sessions[sess.Id()] = sess
		return
	}
}

// setMemcacheSession sets the specified session in the Memcache.
func (s *memcacheStore) setMemcacheSession(sess Session) (success bool) {
	item := &memcache.Item{
		Key:        s.keyPrefix + sess.Id(),
		Object:     sess,
		Expiration: sess.Timeout(),
	}

	var err error
	for i := 0; i < s.retries; i++ {
		if err = s.codec.Set(s.ctx, item); err == nil {
			return true
		}
	}

	s.ctx.Errorf("Failed to add session to memcache, id: %s, error: %v", sess.Id(), err)
	return false
}

// Remove is to implement Store.Remove().
func (s *memcacheStore) Remove(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	var err error
	for i := 0; i < s.retries; i++ {
		if err = memcache.Delete(s.ctx, s.keyPrefix+sess.Id()); err == nil || err == memcache.ErrCacheMiss {
			s.ctx.Infof("Session removed: %s", sess.Id())
			delete(s.sessions, sess.Id())
			if !s.onlyMemcache {
				// Also from the Datastore:
				key := datastore.NewKey(s.ctx, s.dsEntityName, sess.Id(), 0, nil)
				datastore.Delete(s.ctx, key) // Omitting error check...
			}
			return
		}
	}
	s.ctx.Errorf("Failed to remove session from memcache, id: %s, error: %v", sess.Id(), err)
}

// Close is to implement Store.Close().
func (s *memcacheStore) Close() {
	// Flush out sessions that were accessed from this store. No need locking, we're closing...
	// We could use Cocec.SetMulti(), but sessions will contain at most 1 session like all the times.
	for _, sess := range s.sessions {
		s.setMemcacheSession(sess)
	}

	if s.onlyMemcache {
		return // Don't save to Datastore
	}

	if s.syncDatastoreSave {
		s.saveToDatastore()
	} else {
		go s.saveToDatastore()
	}
}

// saveToDatastore saves the sessions of the Store to the Datastore
// in the caller's goroutine.
func (s *memcacheStore) saveToDatastore() {
	// Save sessions that were accessed from this store. No need locking, we're closing...
	// We could use datastore.PutMulti(), but sessions will contain at most 1 session like all the times.
	for _, sess := range s.sessions {
		value, err := s.codec.Marshal(sess)
		if err != nil {
			s.ctx.Errorf("Failed to marshal session: %s, error: %v", sess.Id(), err)
			continue
		}
		e := SessEntity{
			Expires: sess.Accessed().Add(sess.Timeout()),
			Value:   value,
		}
		key := datastore.NewKey(s.ctx, s.dsEntityName, sess.Id(), 0, nil)
		for i := 0; i < s.retries; i++ {
			if _, err = datastore.Put(s.ctx, key, &e); err == nil {
				break
			}
		}
		if err != nil {
			s.ctx.Errorf("Failed to save session to datastore: %s, error: %v", sess.Id(), err)
		}
	}
}

// TODO
func PurgeExpiredSessionsFromDS() bool {
	return false
}
