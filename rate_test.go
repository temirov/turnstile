package main

import (
	"testing"
	"time"
)

func TestWindowLimiter_AllowsThenBlocksThenResets(t *testing.T) {
	limiter := &windowLimiter{
		windowEnd:    timeNow().Unix() + 60,
		counts:       make(map[string]int),
		perMinuteCap: 2,
	}
	key := "http://example|127.0.0.1"

	if !limiter.allow(key) || !limiter.allow(key) {
		t.Fatalf("expected first two to pass")
	}
	if limiter.allow(key) {
		t.Fatalf("expected third to fail")
	}

	// Force new window and try again
	limiter.windowEnd = timeNow().Unix() - 1
	if !limiter.allow(key) {
		t.Fatalf("expected after window reset to pass")
	}
}

func TestReplayStore_MarkAndRejectDuplicate(t *testing.T) {
	store := &replayStore{seen: make(map[string]int64)}
	now := timeNow()
	if !store.mark("abc", now.Add(5*time.Minute)) {
		t.Fatalf("first mark should pass")
	}
	if store.mark("abc", now.Add(5*time.Minute)) {
		t.Fatalf("duplicate should fail")
	}
	// expire
	store.seen["old"] = now.Add(-1 * time.Minute).Unix()
	if !store.mark("new", now.Add(5*time.Minute)) {
		t.Fatalf("new id should pass")
	}
}
