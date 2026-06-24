//go:build !darwin

package updater

func finalizeUpdate(string) error {
	return nil
}

func applyArchiveUpdate([]byte, string, string) (bool, error) {
	return false, nil
}
