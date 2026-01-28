package tests

import (
	"testing"
	"time"

	"zckyachmd/lifeline/internal/security/confirm"
)

func TestConfirmSingleUse(t *testing.T) {
	mgr := confirm.New(50 * time.Millisecond)
	token, _ := mgr.Issue(1, "cmd", []string{"a"}, false)

	if _, err := mgr.Consume(1, token); err != nil {
		t.Fatalf("consume failed: %v", err)
	}
	if _, err := mgr.Consume(1, token); err == nil {
		t.Fatalf("expected second consume to fail")
	}
}

func TestConfirmExpiry(t *testing.T) {
	mgr := confirm.New(10 * time.Millisecond)
	token, _ := mgr.Issue(1, "cmd", nil, false)
	time.Sleep(20 * time.Millisecond)
	if _, err := mgr.Consume(1, token); err == nil {
		t.Fatalf("expected expiry")
	}
}
