package dpop

import (
	"sync"
	"time"
)

// ReplayStore tracks previously accepted DPoP proof identifiers.
// ReplayStoreは過去に受理したDPoP Proof Identifierを管理する。
type ReplayStore interface {
	CheckAndStore(jkt string, jti string) bool
}

// MemoryReplayStore provides in-memory replay detection.
// MemoryReplayStoreはIn-MemoryでReplay Detectionを行う。
type MemoryReplayStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
	ttl     time.Duration
	now     func() time.Time
}

// NewMemoryReplayStore creates an in-memory replay store.
// NewMemoryReplayStoreはIn-Memory Replay Storeを作成する。
func NewMemoryReplayStore(ttl time.Duration) *MemoryReplayStore {
	return &MemoryReplayStore{
		entries: make(map[string]time.Time),
		ttl:     ttl,
		now:     time.Now,
	}
}

// CheckAndStore returns false if the proof was already seen.
// CheckAndStoreはProofが既に使用済みの場合falseを返す。
func (s *MemoryReplayStore) CheckAndStore(
	jkt string,
	jti string,
) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()

	for key, expiresAt := range s.entries {
		if !expiresAt.After(now) {
			delete(s.entries, key)
		}
	}

	key := jkt + ":" + jti

	if expiresAt, exists := s.entries[key]; exists &&
		expiresAt.After(now) {
		return false
	}

	s.entries[key] = now.Add(s.ttl)

	return true
}
