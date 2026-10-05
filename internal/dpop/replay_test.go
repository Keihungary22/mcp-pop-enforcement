package dpop

import (
	"testing"
	"time"
)

func TestMemoryReplayStoreRejectsReplay(t *testing.T) {
	store := NewMemoryReplayStore(5 * time.Minute)

	if !store.CheckAndStore("key-a", "jti-1") {
		t.Fatal("expected first proof to be accepted")
	}

	if store.CheckAndStore("key-a", "jti-1") {
		t.Fatal("expected replayed proof to be rejected")
	}
}

func TestMemoryReplayStoreSeparatesKeys(t *testing.T) {
	store := NewMemoryReplayStore(5 * time.Minute)

	if !store.CheckAndStore("key-a", "same-jti") {
		t.Fatal("expected first proof to be accepted")
	}

	if !store.CheckAndStore("key-b", "same-jti") {
		t.Fatal("expected same JTI from another key to be accepted")
	}
}

func TestMemoryReplayStoreExpiresEntries(t *testing.T) {
	now := time.Now()

	store := NewMemoryReplayStore(5 * time.Minute)
	store.now = func() time.Time {
		return now
	}

	if !store.CheckAndStore("key-a", "jti-1") {
		t.Fatal("expected first proof to be accepted")
	}

	now = now.Add(6 * time.Minute)

	if !store.CheckAndStore("key-a", "jti-1") {
		t.Fatal("expected expired replay entry to be accepted again")
	}
}
