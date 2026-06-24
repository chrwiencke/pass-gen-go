package updater

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var pendingRelaunch pendingRelaunchState

type pendingRelaunchState struct {
	mu     sync.Mutex
	script string
	args   []string
}

func finalizeUpdate(targetPath string) error {
	appDir, ok := appBundleRoot(targetPath)
	if !ok {
		return nil
	}

	if err := verifyCodeSignature(appDir); err == nil {
		return nil
	}

	cmd := exec.Command("/usr/bin/codesign", "--force", "--deep", "--sign", "-", "--timestamp=none", appDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("re-sign updated app bundle: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func applyArchiveUpdate(body []byte, assetName, targetPath string) (bool, error) {
	if !isZipAsset(assetName) || !isMacOSArchive(assetName) {
		return false, nil
	}

	currentAppDir, ok := appBundleRoot(targetPath)
	if !ok {
		return false, nil
	}

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return true, fmt.Errorf("open update archive %q: %w", assetName, err)
	}

	appRoot, err := archiveAppRoot(reader.File)
	if err != nil {
		return true, fmt.Errorf("inspect update archive %q: %w", assetName, err)
	}

	parentDir := filepath.Dir(currentAppDir)
	tempDir, err := os.MkdirTemp(parentDir, ".gopass-update-*")
	if err != nil {
		return true, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(tempDir)
		}
	}()

	nextAppDir := filepath.Join(tempDir, filepath.Base(currentAppDir))
	if err := extractAppBundle(reader.File, appRoot, nextAppDir); err != nil {
		return true, err
	}
	if err := verifyCodeSignature(nextAppDir); err != nil {
		return true, err
	}

	backupDir := filepath.Join(tempDir, filepath.Base(currentAppDir)+".previous")
	scriptPath := filepath.Join(tempDir, "finish-update.sh")
	if err := writeRelaunchScript(scriptPath); err != nil {
		return true, err
	}

	pendingRelaunch.set(scriptPath,
		strconv.Itoa(os.Getpid()),
		currentAppDir,
		nextAppDir,
		backupDir,
		tempDir,
	)
	cleanup = false
	return true, nil
}

func StartPendingRelaunch() (bool, error) {
	return pendingRelaunch.start()
}

func (s *pendingRelaunchState) set(script string, args ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.script = script
	s.args = append([]string(nil), args...)
}

func (s *pendingRelaunchState) start() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.script == "" {
		return false, nil
	}

	cmd := exec.Command(s.script, s.args...)
	if err := cmd.Start(); err != nil {
		return true, err
	}
	s.script = ""
	s.args = nil
	return true, nil
}

func writeRelaunchScript(path string) error {
	const script = `#!/bin/sh
set -eu

pid="$1"
current_app="$2"
next_app="$3"
backup_app="$4"
temp_dir="$5"

while kill -0 "$pid" 2>/dev/null; do
	sleep 0.2
done

rm -rf "$backup_app"
if [ -e "$current_app" ]; then
	mv "$current_app" "$backup_app"
fi

if ! mv "$next_app" "$current_app"; then
	if [ -e "$backup_app" ]; then
		mv "$backup_app" "$current_app"
	fi
	exit 1
fi

/usr/bin/open "$current_app"
rm -rf "$backup_app"
rm -rf "$temp_dir"
`
	return os.WriteFile(path, []byte(script), 0o755)
}

func verifyCodeSignature(appDir string) error {
	cmd := exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", appDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify updated app bundle: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func appBundleRoot(targetPath string) (string, bool) {
	macOSDir := filepath.Dir(filepath.Clean(targetPath))
	if filepath.Base(macOSDir) != "MacOS" {
		return "", false
	}
	contentsDir := filepath.Dir(macOSDir)
	if filepath.Base(contentsDir) != "Contents" {
		return "", false
	}
	appDir := filepath.Dir(contentsDir)
	if filepath.Ext(appDir) != ".app" {
		return "", false
	}
	return appDir, true
}

func isMacOSArchive(assetName string) bool {
	name := strings.ToLower(assetName)
	return strings.Contains(name, "macos") || strings.Contains(name, "darwin")
}

func archiveAppRoot(files []*zip.File) (string, error) {
	var appRoot string
	for _, file := range files {
		name := cleanArchivePath(file.Name)
		if name == "" {
			continue
		}
		parts := strings.Split(name, "/")
		for i, part := range parts {
			if filepath.Ext(part) != ".app" {
				continue
			}
			root := strings.Join(parts[:i+1], "/")
			if appRoot != "" && appRoot != root {
				return "", fmt.Errorf("contains multiple app bundles")
			}
			appRoot = root
			break
		}
	}
	if appRoot == "" {
		return "", fmt.Errorf("does not contain an app bundle")
	}
	return appRoot, nil
}

func extractAppBundle(files []*zip.File, appRoot, targetAppDir string) error {
	prefix := appRoot + "/"
	for _, file := range files {
		name := cleanArchivePath(file.Name)
		if name != appRoot && !strings.HasPrefix(name, prefix) {
			continue
		}

		rel := strings.TrimPrefix(name, prefix)
		if rel == "" {
			continue
		}
		targetPath := filepath.Join(targetAppDir, filepath.FromSlash(rel))
		if !strings.HasPrefix(targetPath, targetAppDir+string(os.PathSeparator)) {
			return fmt.Errorf("archive contains unsafe path %q", file.Name)
		}

		mode := file.FileInfo().Mode()
		if mode.IsDir() {
			if err := os.MkdirAll(targetPath, mode.Perm()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		if mode&os.ModeSymlink != 0 {
			linkTarget, err := readZipFile(file)
			if err != nil {
				return err
			}
			if filepath.IsAbs(string(linkTarget)) {
				return fmt.Errorf("archive contains absolute symlink %q", file.Name)
			}
			if err := os.Symlink(string(linkTarget), targetPath); err != nil {
				return err
			}
			continue
		}
		if !mode.IsRegular() {
			continue
		}

		contents, err := readZipFile(file)
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, contents, mode.Perm()); err != nil {
			return err
		}
	}
	return nil
}

func cleanArchivePath(name string) string {
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return ""
	}
	return clean
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
