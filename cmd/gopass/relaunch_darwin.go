package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"gopass/internal/updater"
)

func relaunchApp() error {
	if handled, err := updater.StartPendingRelaunch(); handled || err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}

	if appDir, ok := appBundleRoot(exe); ok {
		return exec.Command("/usr/bin/open", appDir).Start()
	}
	return exec.Command(exe).Start()
}

func appBundleRoot(exe string) (string, bool) {
	macOSDir := filepath.Dir(filepath.Clean(exe))
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
