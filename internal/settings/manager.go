package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	PasteShortcut string              `json:"pasteShortcut"`
	Templates     []password.Template `json:"templates,omitempty"`
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
	s.Templates = normalizeTemplates(s.Templates)
	return s
}

func (s Settings) Validate() error {
	if err := s.Settings.Validate(); err != nil {
		return err
	}
	if err := validateTemplates(s.Templates); err != nil {
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

func (m *Manager) Templates() []password.Template {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneTemplates(m.value.Templates)
}

func (m *Manager) Save(nextPassword password.Settings, pasteShortcut string) error {
	m.mu.RLock()
	templates := cloneTemplates(m.value.Templates)
	m.mu.RUnlock()

	next := Settings{
		Settings:      nextPassword,
		PasteShortcut: pasteShortcut,
		Templates:     templates,
	}
	return m.saveSettings(next)
}

func (m *Manager) SaveTemplate(name string, templateSettings password.Settings) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("template name is required")
	}

	m.mu.RLock()
	next := m.value
	next.Templates = cloneTemplates(next.Templates)
	m.mu.RUnlock()

	templateSettings = templateSettings.Normalize()
	if err := templateSettings.Validate(); err != nil {
		return err
	}

	replacement := password.Template{Name: name, Settings: templateSettings}
	for i, template := range next.Templates {
		if template.Name == name {
			next.Templates[i] = replacement
			return m.saveSettings(next)
		}
	}

	next.Templates = append(next.Templates, replacement)
	return m.saveSettings(next)
}

func (m *Manager) DeleteTemplate(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("template name is required")
	}

	m.mu.RLock()
	next := m.value
	next.Templates = cloneTemplates(next.Templates)
	m.mu.RUnlock()

	filtered := next.Templates[:0]
	found := false
	for _, template := range next.Templates {
		if template.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, template)
	}
	if !found {
		return fmt.Errorf("template %q was not found", name)
	}

	next.Templates = filtered
	return m.saveSettings(next)
}

func (m *Manager) saveSettings(next Settings) error {
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

func normalizeTemplates(templates []password.Template) []password.Template {
	if len(templates) == 0 {
		return nil
	}

	normalized := make([]password.Template, 0, len(templates))
	for _, template := range templates {
		template.Name = strings.TrimSpace(template.Name)
		template.Settings = template.Settings.Normalize()
		normalized = append(normalized, template)
	}
	return normalized
}

func validateTemplates(templates []password.Template) error {
	seen := map[string]bool{}
	for _, template := range templates {
		if template.Name == "" {
			return fmt.Errorf("template name is required")
		}
		if seen[template.Name] {
			return fmt.Errorf("template %q is duplicated", template.Name)
		}
		seen[template.Name] = true
		if err := template.Settings.Validate(); err != nil {
			return fmt.Errorf("template %q: %w", template.Name, err)
		}
	}
	return nil
}

func cloneTemplates(templates []password.Template) []password.Template {
	if len(templates) == 0 {
		return nil
	}
	cloned := make([]password.Template, len(templates))
	copy(cloned, templates)
	return cloned
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
