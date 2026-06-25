//go:build !darwin

package accessibility

func RequestPermission() bool {
	return true
}
