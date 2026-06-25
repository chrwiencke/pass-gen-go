package settings

import (
	"path/filepath"
	"testing"

	"gopass/internal/password"
)

func TestSaveTemplateAddsReplacesAndDeletesTemplates(t *testing.T) {
	manager := &Manager{
		path:  filepath.Join(t.TempDir(), settingsFile),
		value: DefaultSettings(),
	}

	randomSettings := password.Settings{
		Mode:      password.ModeRandom,
		Language:  password.LanguageEnglish,
		MinLength: 8,
		MaxLength: 8,
		Lowercase: true,
		Uppercase: true,
		Numbers:   true,
		Special:   true,
	}
	if err := manager.SaveTemplate("Short random", randomSettings); err != nil {
		t.Fatalf("SaveTemplate() returned error: %v", err)
	}

	templates := manager.Templates()
	if len(templates) != 1 {
		t.Fatalf("Templates() returned %d templates, want 1", len(templates))
	}
	if templates[0].Name != "Short random" || templates[0].Settings.MinLength != 8 {
		t.Fatalf("template = %+v, want Short random with min length 8", templates[0])
	}

	randomSettings.MinLength = 12
	randomSettings.MaxLength = 12
	if err := manager.SaveTemplate("Short random", randomSettings); err != nil {
		t.Fatalf("SaveTemplate() replacement returned error: %v", err)
	}

	templates = manager.Templates()
	if len(templates) != 1 {
		t.Fatalf("Templates() returned %d templates after replacement, want 1", len(templates))
	}
	if templates[0].Settings.MinLength != 12 {
		t.Fatalf("replacement min length = %d, want 12", templates[0].Settings.MinLength)
	}

	if err := manager.DeleteTemplate("Short random"); err != nil {
		t.Fatalf("DeleteTemplate() returned error: %v", err)
	}
	if templates := manager.Templates(); len(templates) != 0 {
		t.Fatalf("Templates() returned %d templates after delete, want 0", len(templates))
	}
}

func TestSavePreservesTemplates(t *testing.T) {
	manager := &Manager{
		path:  filepath.Join(t.TempDir(), settingsFile),
		value: DefaultSettings(),
	}

	templateSettings := password.DefaultSettings()
	templateSettings.Language = password.LanguageEnglish
	if err := manager.SaveTemplate("English", templateSettings); err != nil {
		t.Fatalf("SaveTemplate() returned error: %v", err)
	}

	mainSettings := password.DefaultSettings()
	mainSettings.Mode = password.ModeRandom
	mainSettings.MinLength = 10
	mainSettings.MaxLength = 10
	if err := manager.Save(mainSettings, manager.PasteShortcut(), manager.AutomaticUpdates()); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	templates := manager.Templates()
	if len(templates) != 1 {
		t.Fatalf("Templates() returned %d templates, want 1", len(templates))
	}
	if templates[0].Name != "English" {
		t.Fatalf("template name = %q, want English", templates[0].Name)
	}
}

func TestAutomaticUpdatesDefaultDisabled(t *testing.T) {
	settings := DefaultSettings()
	if settings.AutomaticUpdates {
		t.Fatal("DefaultSettings().AutomaticUpdates = true, want false")
	}

	manager := &Manager{value: settings}
	if manager.AutomaticUpdates() {
		t.Fatal("AutomaticUpdates() = true, want false")
	}
}

func TestSavePersistsAutomaticUpdates(t *testing.T) {
	manager := &Manager{
		path:  filepath.Join(t.TempDir(), settingsFile),
		value: DefaultSettings(),
	}

	if err := manager.Save(password.DefaultSettings(), manager.PasteShortcut(), true); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	if !manager.AutomaticUpdates() {
		t.Fatal("AutomaticUpdates() = false, want true")
	}
}
