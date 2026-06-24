//go:build !darwin && !windows

package paste

import "fmt"

func Send() error {
	return fmt.Errorf("paste shortcut is only implemented for macOS and Windows")
}
