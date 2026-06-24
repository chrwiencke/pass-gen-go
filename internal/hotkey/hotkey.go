package hotkey

import "sync"

type Manager struct {
	mu       sync.Mutex
	callback func()
	current  string
}

func New(callback func()) *Manager {
	return &Manager{callback: callback}
}

func (m *Manager) SetShortcut(value string) error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if value == m.current {
		return nil
	}
	if err := register(value, m.callback); err != nil {
		return err
	}
	m.current = value
	return nil
}

func (m *Manager) Close() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	unregister()
	m.current = ""
}
