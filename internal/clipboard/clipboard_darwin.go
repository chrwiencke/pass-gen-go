package clipboard

import (
	"os/exec"
	"strings"
)

// CopyText copies text to the macOS clipboard using the system pbcopy command.
func CopyText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
