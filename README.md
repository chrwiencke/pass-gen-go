# GoPass

A small Go menu-bar/taskbar tray app for macOS and Windows.

Left-click the key icon in the macOS menu bar or Windows tray to generate and copy a password. The default settings generate a Norwegian passphrase like:

```text
Fjell-Ovenfor3
```

Default rules:

- `Word-WordDigit` format, e.g. each word starts with one capital letter and the rest is lowercase
- Norwegian words, written with plain ASCII only
- no `æ`, `ø`, or `å`
- length: over 14 and under 22 characters, meaning 15–21 characters
- random choices use `crypto/rand`
- the generated password is not logged, displayed, or stored by the app

The tray menu also has a `Settings...` item. It opens a native Fyne settings window where you can choose:

- passphrase or random password
- major European passphrase languages: Czech, Danish, Dutch, English, Finnish, French, German, Hungarian, Italian, Norwegian, Polish, Portuguese, Romanian, Spanish, and Swedish
- minimum and maximum length
- lowercase, uppercase, numbers, and special characters
- the global shortcut that generates a new password and pastes it into the active app

The default paste shortcut is `Ctrl+Command+P` on macOS and `Ctrl+Windows+P` on Windows. Settings are saved as JSON in the user's OS config directory and are loaded again the next time GoPass starts. Passphrase words are stored as plain ASCII; accents are removed where needed so generated passwords stay compatible with strict password fields.

On macOS, GoPass may need Accessibility permission before it can paste into another app.

## Important note about “one app”

This is one Go codebase for both macOS and Windows. macOS and Windows still require separate compiled binaries because an `.app` bundle cannot run on Windows and a `.exe` cannot run on macOS.

## Requirements

- Go 1.25 or newer
- Internet access the first time you build, so Go can download dependencies

Dependencies:

- `fyne.io/fyne/v2` for the settings window
- `fyne.io/systray` for the cross-platform tray/menu-bar icon
- `github.com/minio/selfupdate` for replacing the running executable during updates

Clipboard copying does not use an external Go clipboard dependency:

- macOS: built-in `pbcopy` command
- Windows: native Win32 clipboard API

This avoids the `_cgo_init` duplicate-symbol linker conflict that can happen when systray and some PureGo clipboard dependencies are linked into the same macOS build.

## First-time setup

```bash
go mod download
# or, if you want Go to refresh go.sum:
go mod tidy
```

## Run during development

```bash
go run ./cmd/gopass
```

Development builds use version `dev`, so they do not check GitHub for updates.

## Build for macOS

From macOS or another machine capable of Go Darwin cross-compilation:

```bash
./scripts/build-macos.sh
```

Output:

```text
dist/macos-<arch>/GoPass.app
dist/macos-<arch>/gopass-darwin-<arch>
dist/macos-<arch>/gopass-darwin-<arch>.sha256
```

On macOS, the bundle enables `LSUIElement`, so it appears in the menu bar without showing a Dock icon.

Set `VERSION` when building a release:

```bash
VERSION=1.2.3 ./scripts/build-macos.sh
```

## Build for Windows

From PowerShell:

```powershell
./scripts/build-windows.ps1
```

Output:

```text
dist/windows-amd64/gopass.exe
dist/windows-amd64/gopass-windows-amd64.exe
dist/windows-amd64/gopass-windows-amd64.exe.sha256
```

The Windows build uses `-H=windowsgui`, so it should not open a console window.

Set `$env:VERSION` when building a release:

```powershell
$env:VERSION = "1.2.3"
./scripts/build-windows.ps1
```

If your editor or a local cross-compile reports that `github.com/go-gl/gl/v2.1/gl`
has no Go files for `windows,amd64`, the Windows target is being checked without
CGO. For a real Windows build, use the PowerShell script above or GitHub Actions;
both enable CGO and use MinGW-w64. For editor diagnostics only, either unset the
Windows target (`GOOS`/`GOARCH`) or add Fyne's in-memory app tag:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -tags ci ./cmd/gopass
```

Do not use that `ci` tag for release builds.

## Build both from macOS/Linux shell

```bash
./scripts/build-all.sh
```

`build-all.sh` requires CGO for systray. It builds macOS locally, and it builds Windows only when `x86_64-w64-mingw32-gcc` is available. If you previously saw a linker error like `duplicated definition of symbol _cgo_init`, update to this version and run:

```bash
go clean -cache
go mod tidy
sh scripts/build-all.sh
```

## Self updates

On launch, GoPass checks the latest release at:

```text
https://github.com/chrwiencke/pass-gen-go/releases/latest
```

The tray menu shows an `Update to <version>` item only when the latest release tag is newer than the app's built-in version and the release contains a current-platform updater asset. The updater prefers the raw binary assets:

```text
gopass-darwin-amd64
gopass-darwin-arm64
gopass-windows-amd64.exe
```

It can also use the zipped installer assets produced by GitHub Actions:

```text
GoPass-macos-amd64.zip
GoPass-macos-arm64.zip
GoPass-windows-amd64.zip
```

Upload the matching `.sha256` file next to each raw binary asset for checksum verification. After the user right-clicks the tray/menu-bar icon and clicks `Update`, the app downloads the release asset, applies it with `github.com/minio/selfupdate`, and asks for a restart so the new binary is used.

## Change the word lists

Edit:

```text
internal/password/words.go
internal/password/words_english.go
internal/password/words_european.go
```

Then run:

```bash
go test ./internal/password
```

The tests check that the default generated passwords still match `Word-WordDigit`, configurable passphrases and random passwords respect their settings, and both word lists contain only lowercase plain ASCII words.
