package mode

import "sync"

// Mode represents bot operation mode.
type Mode string

const (
	Emergency Mode = "emergency"
	ReadOnly  Mode = "readonly"
	Lockdown  Mode = "lockdown"
)

// Manager manages current mode safely.
type Manager struct {
	current Mode
	mu      sync.RWMutex
}

// New creates a manager with initial mode.
func New(initial Mode) *Manager {
	return &Manager{current: initial}
}

// Current returns current mode.
func (m *Manager) Current() Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Set updates mode.
func (m *Manager) Set(next Mode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = next
}

// Allowed checks whether desired action requiring a minimum mode is permitted.
// Requirement: emergency > readonly > lockdown hierarchy.
func (m *Manager) Allowed(required Mode) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return rank(m.current) >= rank(required)
}

func rank(md Mode) int {
	switch md {
	case Emergency:
		return 3
	case ReadOnly:
		return 2
	case Lockdown:
		return 1
	default:
		return 0
	}
}
