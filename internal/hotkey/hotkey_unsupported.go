//go:build !darwin && !windows

package hotkey

import "fmt"

func register(value string, callback func()) error {
	return fmt.Errorf("global shortcuts are only implemented for macOS and Windows")
}

func unregister() {}
