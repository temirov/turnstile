package main

import (
	"sync"
	"time"
)

type replayStore struct {
	mutex sync.Mutex
	seen  map[string]int64
}

func (store *replayStore) mark(tokenID string, expirationTime time.Time) bool {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	currentUnix := time.Now().Unix()
	for cachedID, cachedExp := range store.seen {
		if cachedExp <= currentUnix {
			delete(store.seen, cachedID)
		}
	}
	if _, exists := store.seen[tokenID]; exists {
		return false
	}
	store.seen[tokenID] = expirationTime.Unix()
	return true
}

type windowLimiter struct {
	mutex        sync.Mutex
	windowEnd    int64
	counts       map[string]int
	perMinuteCap int
}

func (limiter *windowLimiter) allow(bucketKey string) bool {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	currentUnix := time.Now().Unix()
	if currentUnix >= limiter.windowEnd {
		limiter.windowEnd = currentUnix + 60
		limiter.counts = make(map[string]int)
	}
	if limiter.counts[bucketKey] >= limiter.perMinuteCap {
		return false
	}
	limiter.counts[bucketKey] = limiter.counts[bucketKey] + 1
	return true
}
