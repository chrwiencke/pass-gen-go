# GoPass

A small Go menu-bar/taskbar tray app for macOS and Windows.

Left-click the key icon in the macOS menu bar or Windows tray to generate and copy a password like:

```text
Fjell-Ovenfor3
```

Rules implemented:

- `Word-WordDigit` format, e.g. each word starts with one capital letter and the rest is lowercase
- Norwegian words, written with plain ASCII only
- no `æ`, `ø`, or `å`
- length: over 14 and under 22 characters, meaning 15–21 characters
- random choices use `crypto/rand`
- the generated password is not logged, displayed, or stored by the app

## Important note about “one app”

This is one Go codebase for both macOS and Windows. macOS and Windows still require separate compiled binaries because an `.app` bundle cannot run on Windows and a `.exe` cannot run on macOS.

## Requirements

- Go 1.23 or newer
- Internet access the first time you build, so Go can download dependencies

Dependencies:

- `github.com/gogpu/systray v0.1.0` for the cross-platform tray/menu-bar icon

Clipboard copying does not use an external Go clipboard dependency:

- macOS: built-in `pbcopy` command
- Windows: native Win32 clipboard API

This avoids the `_cgo_init` duplicate-symbol linker conflict that can happen when `github.com/gogpu/systray` and some PureGo clipboard dependencies are linked into the same macOS build.

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

## Build for macOS

From macOS or another machine capable of Go Darwin cross-compilation:

```bash
./scripts/build-macos.sh
```

Output:

```text
dist/macos-<arch>/GoPass.app
```

On macOS, the bundle uses `LSUIElement=1`, so it appears in the menu bar without showing a Dock icon.

## Build for Windows

From PowerShell:

```powershell
./scripts/build-windows.ps1
```

Output:

```text
dist/windows-amd64/gopass.exe
```

The Windows build uses `-H=windowsgui`, so it should not open a console window.

## Build both from macOS/Linux shell

```bash
./scripts/build-all.sh
```

`build-all.sh` keeps `CGO_ENABLED=0` for both targets. If you previously saw a linker error like `duplicated definition of symbol _cgo_init`, update to this version and run:

```bash
go clean -cache
go mod tidy
sh scripts/build-all.sh
```

## Change the word list

Edit:

```text
internal/password/words.go
```

Then run:

```bash
go test ./internal/password
```

The tests check that generated passwords match `Word-WordDigit`, are 15–21 characters long, contain only plain ASCII, and that the word list has at least 1,000 lowercase entries. The included list currently has 1,224 entries.
