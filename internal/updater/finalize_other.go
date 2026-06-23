//go:build !darwin

package updater

func finalizeUpdate(string) error {
	return nil
}
