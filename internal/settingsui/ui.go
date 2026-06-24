package settingsui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"gopass/internal/password"
	appsettings "gopass/internal/settings"
	"gopass/internal/shortcut"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	modePassphraseLabel = "Passphrase"
	modeRandomLabel     = "Random password"
)

type UI struct {
	app     fyne.App
	manager *appsettings.Manager
	onSave  func()

	mu     sync.Mutex
	window fyne.Window
	form   *settingsForm
}

type settingsForm struct {
	mode      *widget.Select
	language  *widget.Select
	minLength *widget.Entry
	maxLength *widget.Entry
	lowercase *widget.Check
	uppercase *widget.Check
	numbers   *widget.Check
	special   *widget.Check
	shortcut  *widget.Entry
	status    *widget.Label
}

func New(app fyne.App, manager *appsettings.Manager, onSave func()) *UI {
	return &UI{
		app:     app,
		manager: manager,
		onSave:  onSave,
	}
}

func (u *UI) Open() error {
	if u.app == nil {
		return fmt.Errorf("fyne app is unavailable")
	}
	if u.manager == nil {
		return fmt.Errorf("settings manager is unavailable")
	}

	fyne.Do(func() {
		u.openOnUIThread()
	})
	return nil
}

func (u *UI) Close() {
	if u == nil || u.app == nil {
		return
	}

	fyne.Do(func() {
		u.closeOnUIThread()
	})
}

func (u *UI) openOnUIThread() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.window == nil {
		u.window, u.form = u.buildWindow()
	}

	u.form.load(u.manager.Current(), u.manager.PasteShortcut())
	u.window.Show()
	u.window.RequestFocus()
}

func (u *UI) closeOnUIThread() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.window == nil {
		return
	}

	u.window.SetCloseIntercept(nil)
	u.window.Close()
	u.window = nil
	u.form = nil
}

func (u *UI) buildWindow() (fyne.Window, *settingsForm) {
	w := u.app.NewWindow("GoPass Settings")
	w.Resize(fyne.NewSize(500, 500))
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	form := newSettingsForm()
	form.mode.OnChanged = func(value string) {
		form.setModeControlsEnabled(modeFromLabel(value))
	}

	saveButton := widget.NewButton("Save", func() {
		settings, pasteShortcut, err := form.settings()
		if err != nil {
			form.status.SetText(err.Error())
			return
		}

		if err := u.manager.Save(settings, pasteShortcut); err != nil {
			form.status.SetText("Could not save settings: " + err.Error())
			return
		}

		form.status.SetText("Settings saved")
		if u.onSave != nil {
			u.onSave()
		}
	})
	saveButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Cancel", func() {
		w.Hide()
	})

	content := container.NewVBox(
		widget.NewCard("Password", "", widget.NewForm(
			widget.NewFormItem("Type", form.mode),
			widget.NewFormItem("Language", form.language),
			widget.NewFormItem("Minimum length", form.minLength),
			widget.NewFormItem("Maximum length", form.maxLength),
		)),
		widget.NewCard("Characters", "", container.NewVBox(
			form.lowercase,
			form.uppercase,
			form.numbers,
			form.special,
		)),
		widget.NewCard("Shortcut", "", widget.NewForm(
			widget.NewFormItem("Generate and paste", form.shortcut),
		)),
		container.NewHBox(layout.NewSpacer(), cancelButton, saveButton),
		form.status,
	)

	w.SetContent(container.NewPadded(content))
	return w, form
}

func newSettingsForm() *settingsForm {
	minLength := widget.NewEntry()
	minLength.SetPlaceHolder(strconv.Itoa(password.MinLength))

	maxLength := widget.NewEntry()
	maxLength.SetPlaceHolder(strconv.Itoa(password.MaxLength))

	pasteShortcut := widget.NewEntry()
	pasteShortcut.SetPlaceHolder(shortcut.Default())

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord

	return &settingsForm{
		mode:      widget.NewSelect([]string{modePassphraseLabel, modeRandomLabel}, nil),
		language:  widget.NewSelect(languageLabels(), nil),
		minLength: minLength,
		maxLength: maxLength,
		lowercase: widget.NewCheck("Lowercase", nil),
		uppercase: widget.NewCheck("Uppercase", nil),
		numbers:   widget.NewCheck("Numbers", nil),
		special:   widget.NewCheck("Special characters", nil),
		shortcut:  pasteShortcut,
		status:    status,
	}
}

func (f *settingsForm) load(settings password.Settings, pasteShortcut string) {
	settings = settings.Normalize()

	f.mode.SetSelected(labelForMode(settings.Mode))
	f.language.SetSelected(labelForLanguage(settings.Language))
	f.minLength.SetText(strconv.Itoa(settings.MinLength))
	f.maxLength.SetText(strconv.Itoa(settings.MaxLength))
	f.lowercase.SetChecked(settings.Lowercase)
	f.uppercase.SetChecked(settings.Uppercase)
	f.numbers.SetChecked(settings.Numbers)
	f.special.SetChecked(settings.Special)
	f.shortcut.SetText(shortcut.Normalize(pasteShortcut))
	f.status.SetText("")
	f.setModeControlsEnabled(settings.Mode)
}

func (f *settingsForm) settings() (password.Settings, string, error) {
	settings := password.Settings{
		Mode:      modeFromLabel(f.mode.Selected),
		Language:  languageFromLabel(f.language.Selected),
		Lowercase: f.lowercase.Checked,
		Uppercase: f.uppercase.Checked,
		Numbers:   f.numbers.Checked,
		Special:   f.special.Checked,
	}

	minLength, err := parseLength("minimum length", f.minLength.Text)
	if err != nil {
		return settings, "", err
	}
	settings.MinLength = minLength

	maxLength, err := parseLength("maximum length", f.maxLength.Text)
	if err != nil {
		return settings, "", err
	}
	settings.MaxLength = maxLength

	settings = settings.Normalize()
	if err := settings.Validate(); err != nil {
		return settings, "", err
	}

	pasteShortcut, err := parseShortcut(f.shortcut.Text)
	if err != nil {
		return settings, "", err
	}

	return settings, pasteShortcut, nil
}

func (f *settingsForm) setModeControlsEnabled(mode password.Mode) {
	if mode == password.ModeRandom {
		f.language.Disable()
		return
	}
	f.language.Enable()
}

func parseShortcut(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = shortcut.Default()
	}

	parsed, err := shortcut.ParseCurrentPlatform(value)
	if err != nil {
		return "", fmt.Errorf("shortcut %s", err)
	}
	return parsed.String(), nil
}

func parseLength(label, value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", label)
	}
	return parsed, nil
}

func labelForMode(mode password.Mode) string {
	if mode == password.ModeRandom {
		return modeRandomLabel
	}
	return modePassphraseLabel
}

func modeFromLabel(label string) password.Mode {
	if label == modeRandomLabel {
		return password.ModeRandom
	}
	return password.ModePassphrase
}

func labelForLanguage(language password.Language) string {
	return password.LabelForLanguage(language)
}

func languageFromLabel(label string) password.Language {
	language, ok := password.LanguageForLabel(label)
	if ok {
		return language
	}
	return password.LanguageNorwegian
}

func languageLabels() []string {
	options := password.SupportedLanguages()
	labels := make([]string, len(options))
	for i, option := range options {
		labels[i] = option.Label
	}
	return labels
}
