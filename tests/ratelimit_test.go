package tests

import (
	"testing"
	"time"

	rl "zckyachmd/lifeline/internal/security/ratelimit"
)

func TestRateLimit(t *testing.T) {
	lim := rl.New(2, time.Second)
	if !lim.Allow(1) || !lim.Allow(1) {
		t.Fatalf("first two should pass")
	}
	if lim.Allow(1) {
		t.Fatalf("third should be blocked")
	}
	time.Sleep(1100 * time.Millisecond)
	if !lim.Allow(1) {
		t.Fatalf("should allow after window")
	}
}
