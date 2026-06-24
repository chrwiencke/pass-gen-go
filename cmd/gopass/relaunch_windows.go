package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func relaunchApp() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	return exec.Command(exe).Start()
}
