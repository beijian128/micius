package seqchecker

import (
	"sync"
	"time"
)

type hashSet interface {
	set(string, time.Duration)
	get(number string) bool
}

type hashSetImp struct {
	s  map[string]struct{}
	mu sync.RWMutex
}

func newHashSetImp() *hashSetImp {
	return &hashSetImp{s: map[string]struct{}{}}
}

func (hs *hashSetImp) set(key string, expireTime time.Duration) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.s[key] = struct{}{}

	time.AfterFunc(expireTime, func() {
		hs.del(key)
	})
}

func (hs *hashSetImp) get(key string) bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	_, exist := hs.s[key]
	return exist
}

func (hs *hashSetImp) del(key string) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	delete(hs.s, key)
}
