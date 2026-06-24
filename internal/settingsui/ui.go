package settingsui

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"gopass/internal/password"
	appsettings "gopass/internal/settings"
	"gopass/internal/shortcut"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
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
	mode           *widget.Select
	language       *widget.Select
	minLength      *widget.Entry
	maxLength      *widget.Entry
	lowercase      *widget.Check
	uppercase      *widget.Check
	numbers        *widget.Check
	special        *widget.Check
	shortcut       *shortcutCapture
	templateSelect *widget.Select
	templateName   *widget.Entry
	status         *widget.Label
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

	u.form.load(u.manager.Current(), u.manager.PasteShortcut(), u.manager.Templates())
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
	form.templateSelect.OnChanged = func(value string) {
		form.templateName.SetText(value)
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

		u.settingsSaved(form, "Settings saved")
	})
	saveButton.Importance = widget.HighImportance

	loadTemplateButton := widget.NewButton("Load", func() {
		template, ok := findTemplate(u.manager.Templates(), form.templateSelect.Selected)
		if !ok {
			form.status.SetText("Choose a saved template to load")
			return
		}

		form.loadPasswordSettings(template.Settings)
		form.templateName.SetText(template.Name)
		form.status.SetText("Template loaded")
	})

	saveTemplateButton := widget.NewButton("Save template", func() {
		settings, err := form.passwordSettings()
		if err != nil {
			form.status.SetText(err.Error())
			return
		}

		name := strings.TrimSpace(form.templateName.Text)
		if err := u.manager.SaveTemplate(name, settings); err != nil {
			form.status.SetText("Could not save template: " + err.Error())
			return
		}

		form.loadTemplates(u.manager.Templates(), name)
		u.settingsSaved(form, "Template saved")
	})

	deleteTemplateButton := widget.NewButton("Delete", func() {
		name := strings.TrimSpace(form.templateSelect.Selected)
		if name == "" {
			name = strings.TrimSpace(form.templateName.Text)
		}

		if err := u.manager.DeleteTemplate(name); err != nil {
			form.status.SetText("Could not delete template: " + err.Error())
			return
		}

		form.templateName.SetText("")
		form.loadTemplates(u.manager.Templates(), "")
		u.settingsSaved(form, "Template deleted")
	})

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
		widget.NewCard("Templates", "", container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Saved template", form.templateSelect),
				widget.NewFormItem("Template name", form.templateName),
			),
			container.NewHBox(loadTemplateButton, saveTemplateButton, deleteTemplateButton),
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

	pasteShortcut := newShortcutCapture()
	pasteShortcut.SetPlaceHolder(shortcut.Default())

	templateName := widget.NewEntry()
	templateName.SetPlaceHolder("Work login")

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord

	return &settingsForm{
		mode:           widget.NewSelect([]string{modePassphraseLabel, modeRandomLabel}, nil),
		language:       widget.NewSelect(languageLabels(), nil),
		minLength:      minLength,
		maxLength:      maxLength,
		lowercase:      widget.NewCheck("Lowercase", nil),
		uppercase:      widget.NewCheck("Uppercase", nil),
		numbers:        widget.NewCheck("Numbers", nil),
		special:        widget.NewCheck("Special characters", nil),
		shortcut:       pasteShortcut,
		templateSelect: widget.NewSelect(nil, nil),
		templateName:   templateName,
		status:         status,
	}
}

func (f *settingsForm) load(settings password.Settings, pasteShortcut string, templates []password.Template) {
	f.loadPasswordSettings(settings)
	f.shortcut.SetText(shortcut.Normalize(pasteShortcut))
	f.loadTemplates(templates, "")
	f.status.SetText("")
}

func (f *settingsForm) loadPasswordSettings(settings password.Settings) {
	settings = settings.Normalize()

	f.mode.SetSelected(labelForMode(settings.Mode))
	f.language.SetSelected(labelForLanguage(settings.Language))
	f.minLength.SetText(strconv.Itoa(settings.MinLength))
	f.maxLength.SetText(strconv.Itoa(settings.MaxLength))
	f.lowercase.SetChecked(settings.Lowercase)
	f.uppercase.SetChecked(settings.Uppercase)
	f.numbers.SetChecked(settings.Numbers)
	f.special.SetChecked(settings.Special)
	f.setModeControlsEnabled(settings.Mode)
}

func (f *settingsForm) settings() (password.Settings, string, error) {
	settings, err := f.passwordSettings()
	if err != nil {
		return settings, "", err
	}

	pasteShortcut, err := parseShortcut(f.shortcut.Text)
	if err != nil {
		return settings, "", err
	}

	return settings, pasteShortcut, nil
}

func (f *settingsForm) passwordSettings() (password.Settings, error) {
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
		return settings, err
	}
	settings.MinLength = minLength

	maxLength, err := parseLength("maximum length", f.maxLength.Text)
	if err != nil {
		return settings, err
	}
	settings.MaxLength = maxLength

	settings = settings.Normalize()
	if err := settings.Validate(); err != nil {
		return settings, err
	}

	return settings, nil
}

func (f *settingsForm) loadTemplates(templates []password.Template, selected string) {
	names := templateNames(templates)
	f.templateSelect.Options = names
	f.templateSelect.SetSelected(selected)
	f.templateSelect.Refresh()
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

func (u *UI) settingsSaved(form *settingsForm, message string) {
	form.status.SetText(message)
	if u.onSave != nil {
		u.onSave()
	}
}

func findTemplate(templates []password.Template, name string) (password.Template, bool) {
	for _, template := range templates {
		if template.Name == name {
			return template, true
		}
	}
	return password.Template{}, false
}

func templateNames(templates []password.Template) []string {
	names := make([]string, len(templates))
	for i, template := range templates {
		names[i] = template.Name
	}
	return names
}

type shortcutCapture struct {
	widget.Entry

	held map[fyne.KeyName]bool
}

var _ desktop.Keyable = (*shortcutCapture)(nil)

func newShortcutCapture() *shortcutCapture {
	c := &shortcutCapture{held: make(map[fyne.KeyName]bool)}
	c.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	c.ExtendBaseWidget(c)
	return c
}

func (c *shortcutCapture) KeyDown(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	c.held[event.Name] = true
	if shortcutValue, ok := c.shortcutFor(event.Name); ok {
		c.SetText(shortcutValue)
		return
	}
	c.showHeldModifiers()
}

func (c *shortcutCapture) KeyUp(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	delete(c.held, event.Name)
}

func (c *shortcutCapture) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	if shortcutValue, ok := c.shortcutFor(event.Name); ok {
		c.SetText(shortcutValue)
	}
}

func (c *shortcutCapture) TypedRune(r rune) {
}

func (c *shortcutCapture) shortcutFor(key fyne.KeyName) (string, bool) {
	keyName := shortcutKeyName(key)
	if keyName == "" {
		return "", false
	}

	parts := c.modifierParts()
	if len(parts) == 0 {
		return "", false
	}

	parts = append(parts, keyName)
	value := strings.Join(parts, "+")
	parsed, err := shortcut.ParseCurrentPlatform(value)
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}

func (c *shortcutCapture) modifierParts() []string {
	parts := make([]string, 0, 4)
	if c.held[desktop.KeyControlLeft] || c.held[desktop.KeyControlRight] {
		parts = append(parts, "Ctrl")
	}
	if c.held[desktop.KeyShiftLeft] || c.held[desktop.KeyShiftRight] {
		parts = append(parts, "Shift")
	}
	if c.held[desktop.KeySuperLeft] || c.held[desktop.KeySuperRight] {
		if runtime.GOOS == "darwin" {
			parts = append(parts, "Command")
		} else {
			parts = append(parts, "Windows")
		}
	}
	return parts
}

func (c *shortcutCapture) showHeldModifiers() {
	parts := c.modifierParts()
	if len(parts) == 0 {
		return
	}
	c.SetText(strings.Join(append(parts, "..."), "+"))
}

func shortcutKeyName(key fyne.KeyName) string {
	value := string(key)
	if len(value) != 1 {
		return ""
	}
	keyRune := value[0]
	if (keyRune >= 'a' && keyRune <= 'z') || (keyRune >= 'A' && keyRune <= 'Z') {
		return strings.ToUpper(value)
	}
	if keyRune >= '0' && keyRune <= '9' {
		return value
	}
	return ""
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
