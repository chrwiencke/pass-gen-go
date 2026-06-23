//go:build !darwin && !windows

package clipboard

import "fmt"

// CopyText returns an error on unsupported platforms. The app is intended for macOS and Windows.
func CopyText(text string) error {
	return fmt.Errorf("clipboard is only implemented for macOS and Windows")
}
