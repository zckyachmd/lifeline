package confirm

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type PendingAction struct {
	UserID  int64
	Command string
	Args    []string
	Expires time.Time
	Double  bool
}

// Manager handles confirmation tokens.
type Manager struct {
	mu     sync.Mutex
	tokens map[string]PendingAction
	ttl    time.Duration
}

// New creates manager with TTL.
func New(ttl time.Duration) *Manager {
	return &Manager{
		tokens: make(map[string]PendingAction),
		ttl:    ttl,
	}
}

// Issue creates a token for a pending action.
func (m *Manager) Issue(userID int64, command string, args []string, double bool) (string, PendingAction) {
	token := randomToken()
	m.mu.Lock()
	defer m.mu.Unlock()
	pa := PendingAction{UserID: userID, Command: command, Args: args, Expires: time.Now().Add(m.ttl), Double: double}
	m.tokens[token] = pa
	return token, pa
}

// Consume validates and removes a token.
func (m *Manager) Consume(userID int64, token string) (PendingAction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pa, ok := m.tokens[token]
	if !ok {
		return PendingAction{}, errors.New("invalid token")
	}
	delete(m.tokens, token)
	if pa.UserID != userID {
		return PendingAction{}, errors.New("token not owned")
	}
	if time.Now().After(pa.Expires) {
		return PendingAction{}, errors.New("token expired")
	}
	return pa, nil
}

// Sweep removes expired tokens.
func (m *Manager) Sweep() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, v := range m.tokens {
		if now.After(v.Expires) {
			delete(m.tokens, k)
		}
	}
}

func randomToken() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "fallbacktoken"
	}
	return hex.EncodeToString(b)
}
