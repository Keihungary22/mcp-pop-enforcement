package dpop

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestFutureDatedProofCannotReplayDuringFreshnessWindow(t *testing.T) {
	initial := time.Now().Truncate(time.Second)
	now := initial

	key := generateDPoPTestKey(t)
	claims := validProofClaims(initial)
	claims.IssuedAt = jwt.NewNumericDate(initial.Add(20 * time.Second))

	proof := signTestProof(
		t, key, jwkFromPrivateKey(key),
		claims, "dpop+jwt", true,
	)

	verifier := NewVerifier(5*time.Minute, 30*time.Second)
	verifier.now = func() time.Time { return now }

	store := NewMemoryReplayStore(5*time.Minute + 30*time.Second)
	store.now = func() time.Time { return now }

	check := func() (bool, error) {
		result, err := verifier.Verify(
			proof, testAccessToken, testMethod, testHTU,
		)
		if err != nil {
			return false, err
		}
		return store.CheckAndStore(result.JKT, result.JTI), nil
	}

	accepted, err := check()
	if err != nil || !accepted {
		t.Fatalf("initial proof rejected: accepted=%v err=%v", accepted, err)
	}

	now = initial.Add(301 * time.Second)

	accepted, err = check()
	if err != nil {
		t.Fatalf("proof should still be fresh: %v", err)
	}
	if accepted {
		t.Fatal("identical proof replay was accepted")
	}

	now = initial.Add(321 * time.Second)

	_, err = check()
	if err == nil {
		t.Fatal("proof should be rejected after its freshness window")
	}
}
