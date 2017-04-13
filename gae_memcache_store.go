// +build appengine

/*

A Google App Engine Memcache session store implementation.

The implementation stores sessions in the Memcache and also saves sessions to the Datastore as a backup
in case data would be removed from the Memcache. This behaviour is optional, Datastore can be disabled completely.
You can also choose whether saving to Datastore happens synchronously (in the same goroutine)
or asynchronously (in another goroutine).

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
	"net/http"
	"sync"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/memcache"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
)

// A Google App Engine Memcache session store implementation.
type memcacheStore struct {
	ctx context.Context // Appengine context used when accessing the Memcache

	keyPrefix string // Prefix to use in front of session ids to construct Memcache key
	retries   int    // Number of retries to perform in case of general Memcache failures

	codec memcache.Codec // Codec used to marshal and unmarshal a Session to a byte slice

	onlyMemcache       bool   // Tells if sessions are not to be saved in Datastore
	asyncDatastoreSave bool   // Tells if saving in Datastore should happen asynchronously, in a new goroutine
	dsEntityName       string // Name of the datastore entity to use to save sessions

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

	// Tells if saving in Datastore should happen asynchronously (in a new goroutine, possibly after returning),
	// if false, session saving in Datastore will happen in the same goroutine, before returning from the request.
	// Asynchronous saving gives smaller latency (and is enough most of the time as Memcache is always checked first);
	// default value is false which means to save sessions in the Datastore in the same goroutine, synchronously
	// Not used if OnlyMemcache=true.
	// FIXME: See https://github.com/icza/session/issues/3
	AsyncDatastoreSave bool

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
func NewMemcacheStore(ctx context.Context) Store {
	return NewMemcacheStoreOptions(ctx, zeroMemcacheStoreOptions)
}

const defaultDSEntityName = "sess_" // Default value of DSEntityName.

// NewMemcacheStoreOptions returns a new, GAE Memcache session Store with the specified options.
//
// Important! Since accessing the Memcache relies on Appengine Context
// which is bound to an http.Request, the returned Store can only be used for the lifetime of a request!
func NewMemcacheStoreOptions(ctx context.Context, o *MemcacheStoreOptions) Store {
	s := &memcacheStore{
		ctx:                ctx,
		keyPrefix:          o.KeyPrefix,
		retries:            o.Retries,
		onlyMemcache:       o.OnlyMemcache,
		asyncDatastoreSave: o.AsyncDatastoreSave,
		dsEntityName:       o.DSEntityName,
		sessions:           make(map[string]Session, 2),
		mux:                &sync.RWMutex{},
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
		s.dsEntityName = defaultDSEntityName
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
			log.Errorf(s.ctx, "Failed to get session from memcache, id: %s, error: %v", id, err)
		}

		// Ok, we didn't get it from Memcache (either was not there or Memcache service is unavailable).
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
			if e.Expires.Before(time.Now()) {
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
		log.Errorf(s.ctx, "Failed to get session from datastore, id: %s, error: %v", id, err)
		return nil
	}

	// Yes! We have it!
	// "Actualize" it, but first, Mutex is not marshaled, so create a new one:
	sess.mux = &sync.RWMutex{}
	sess.Access()
	s.sessions[id] = sess
	return sess
}

// Add is to implement Store.Add().
func (s *memcacheStore) Add(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.setMemcacheSession(sess) {
		log.Infof(s.ctx, "Session added: %s", sess.ID())
		s.sessions[sess.ID()] = sess
		return
	}
}

// setMemcacheSession sets the specified session in the Memcache.
func (s *memcacheStore) setMemcacheSession(sess Session) (success bool) {
	item := &memcache.Item{
		Key:        s.keyPrefix + sess.ID(),
		Object:     sess,
		Expiration: sess.Timeout(),
	}

	var err error
	for i := 0; i < s.retries; i++ {
		if err = s.codec.Set(s.ctx, item); err == nil {
			return true
		}
	}

	log.Errorf(s.ctx, "Failed to add session to memcache, id: %s, error: %v", sess.ID(), err)
	return false
}

// Remove is to implement Store.Remove().
func (s *memcacheStore) Remove(sess Session) {
	s.mux.Lock()
	defer s.mux.Unlock()

	var err error
	for i := 0; i < s.retries; i++ {
		if err = memcache.Delete(s.ctx, s.keyPrefix+sess.ID()); err == nil || err == memcache.ErrCacheMiss {
			log.Infof(s.ctx, "Session removed: %s", sess.ID())
			delete(s.sessions, sess.ID())
			if !s.onlyMemcache {
				// Also from the Datastore:
				key := datastore.NewKey(s.ctx, s.dsEntityName, sess.ID(), 0, nil)
				datastore.Delete(s.ctx, key) // Omitting error check...
			}
			return
		}
	}
	log.Errorf(s.ctx, "Failed to remove session from memcache, id: %s, error: %v", sess.ID(), err)
}

// Close is to implement Store.Close().
func (s *memcacheStore) Close() {
	// Flush out sessions that were accessed from this store. No need locking, we're closing...
	// We could use Codec.SetMulti(), but sessions will contain at most 1 session like all the times.
	for _, sess := range s.sessions {
		s.setMemcacheSession(sess)
	}

	if s.onlyMemcache {
		return // Don't save to Datastore
	}

	if s.asyncDatastoreSave {
		go s.saveToDatastore()
	} else {
		s.saveToDatastore()
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
			log.Errorf(s.ctx, "Failed to marshal session: %s, error: %v", sess.ID(), err)
			continue
		}
		e := SessEntity{
			Expires: sess.Accessed().Add(sess.Timeout()),
			Value:   value,
		}
		key := datastore.NewKey(s.ctx, s.dsEntityName, sess.ID(), 0, nil)
		for i := 0; i < s.retries; i++ {
			if _, err = datastore.Put(s.ctx, key, &e); err == nil {
				break
			}
		}
		if err != nil {
			log.Errorf(s.ctx, "Failed to save session to datastore: %s, error: %v", sess.ID(), err)
		}
	}
}

// PurgeExpiredSessFromDSFunc returns a request handler function which deletes expired sessions
// from the Datastore.
// dsEntityName is the name of the entity used for saving sessions; pass an empty string
// to use the default value (which is "sess_").
//
// It is recommended to register the returned handler function to a path which then can be defined
// as a cron job to be called periodically, e.g. in every 30 minutes or so (your choice).
// As cron handlers may run up to 10 minutes, the returned handler will stop at 8 minutes
// to complete safely even if there are more expired, undeleted sessions.
//
// The response of the handler func is a JSON text telling if the handler was able to delete all expired sessions,
// or that it was finished early due to the time. Examle of a respone where all expired sessions were deleted:
//
//     {"completed":true}
func PurgeExpiredSessFromDSFunc(dsEntityName string) http.HandlerFunc {
	if dsEntityName == "" {
		dsEntityName = defaultDSEntityName
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		// Delete in batches of 100
		q := datastore.NewQuery(dsEntityName).Filter("exp<", time.Now()).KeysOnly().Limit(100)

		deadline := time.Now().Add(time.Minute * 8)

		for {
			var err error
			var keys []*datastore.Key

			if keys, err = q.GetAll(c, nil); err != nil {
				// Datastore error.
				log.Errorf(c, "Failed to query expired sessions: %v", err)
				http.Error(w, "Failed to query expired sessions!", http.StatusInternalServerError)
			}
			if len(keys) == 0 {
				// We're done, no more expired sessions
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"completed":true}`))
				return
			}

			if err = datastore.DeleteMulti(c, keys); err != nil {
				log.Errorf(c, "Error while deleting expired sessions: %v", err)
			}

			if time.Now().After(deadline) {
				// Our time is up, return
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"completed":false}`))
				return
			}
			// We have time to continue
		}
	}
}
