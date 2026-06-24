package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopass/internal/password"
	"gopass/internal/shortcut"
)

const (
	configDirName  = "GoPass"
	settingsFile   = "settings.json"
	tempFileSuffix = ".tmp"
)

type Manager struct {
	mu    sync.RWMutex
	path  string
	value Settings
}

type Settings struct {
	password.Settings
	PasteShortcut string `json:"pasteShortcut"`
}

func DefaultSettings() Settings {
	return Settings{
		Settings:      password.DefaultSettings(),
		PasteShortcut: shortcut.Default(),
	}
}

func (s Settings) Normalize() Settings {
	s.Settings = s.Settings.Normalize()
	s.PasteShortcut = shortcut.Normalize(s.PasteShortcut)
	return s
}

func (s Settings) Validate() error {
	if err := s.Settings.Validate(); err != nil {
		return err
	}
	_, err := shortcut.ParseCurrentPlatform(s.PasteShortcut)
	return err
}

func NewManager() (*Manager, error) {
	value := DefaultSettings()

	path, err := defaultPath()
	if err != nil {
		return &Manager{value: value}, err
	}

	manager := &Manager{
		path:  path,
		value: value,
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return manager, nil
	}
	if err != nil {
		return manager, err
	}

	var loaded Settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return manager, fmt.Errorf("load settings: %w", err)
	}

	loaded = loaded.Normalize()
	if err := loaded.Validate(); err != nil {
		return manager, fmt.Errorf("load settings: %w", err)
	}

	manager.value = loaded
	return manager, nil
}

func (m *Manager) Current() password.Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.value.Settings
}

func (m *Manager) PasteShortcut() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.value.PasteShortcut
}

func (m *Manager) Save(nextPassword password.Settings, pasteShortcut string) error {
	next := Settings{
		Settings:      nextPassword,
		PasteShortcut: pasteShortcut,
	}
	next = next.Normalize()
	if err := next.Validate(); err != nil {
		return err
	}
	if m.path == "" {
		return fmt.Errorf("settings path is unavailable")
	}

	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}

	tmpPath := m.path + tempFileSuffix
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	if err := replaceFile(tmpPath, m.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	m.mu.Lock()
	m.value = next
	m.mu.Unlock()
	return nil
}

func replaceFile(tmpPath, targetPath string) error {
	if err := os.Rename(tmpPath, targetPath); err == nil {
		return nil
	}

	removeErr := os.Remove(targetPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}

	return os.Rename(tmpPath, targetPath)
}

func (m *Manager) Path() string {
	return m.path
}

func defaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, configDirName, settingsFile), nil
}
