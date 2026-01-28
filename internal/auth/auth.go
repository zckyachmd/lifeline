package auth

import "sync"

// Authorizer enforces Telegram user allowlist.
type Authorizer struct {
	allowed map[int64]struct{}
	mu      sync.RWMutex
}

// New returns Authorizer with provided chat IDs.
func New(ids []int64) *Authorizer {
	a := &Authorizer{allowed: make(map[int64]struct{}, len(ids))}
	for _, id := range ids {
		a.allowed[id] = struct{}{}
	}
	return a
}

// IsAllowed returns true when userID is whitelisted.
func (a *Authorizer) IsAllowed(userID int64) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.allowed[userID]
	return ok
}

// Add adds a new id (not used runtime but testable).
func (a *Authorizer) Add(id int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.allowed[id] = struct{}{}
}
