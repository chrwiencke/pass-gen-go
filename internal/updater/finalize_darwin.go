package updater

import (
	"fmt"
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
