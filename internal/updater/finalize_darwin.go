package updater

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
	defer os.RemoveAll(tempDir)

	nextAppDir := filepath.Join(tempDir, filepath.Base(currentAppDir))
	if err := extractAppBundle(reader.File, appRoot, nextAppDir); err != nil {
		return true, err
	}
	if err := verifyCodeSignature(nextAppDir); err != nil {
		return true, err
	}

	backupDir := filepath.Join(tempDir, filepath.Base(currentAppDir)+".previous")
	if err := os.Rename(currentAppDir, backupDir); err != nil {
		return true, err
	}

	if err := os.Rename(nextAppDir, currentAppDir); err != nil {
		_ = os.Rename(backupDir, currentAppDir)
		return true, err
	}
	return true, nil
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
