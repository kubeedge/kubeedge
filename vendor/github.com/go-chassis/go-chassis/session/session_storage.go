package session

import (
	"time"
)

// Save for setting the session uuid, endpoint, timeout
func Save(sid string, ep string, timeOut time.Duration) {
	Cache.Set(sid, ep, timeOut)
}

// Get return endpoint based on session uuid
func Get(sid string) (ep interface{}, ok bool) {
	ep, ok = Cache.Get(sid)
	return
}

//ClearExpired delete all expired session
func ClearExpired() {
	Cache.DeleteExpired()
}

// Delete delete the session uuid
func Delete(sid string) {
	Cache.Delete(sid)
}
