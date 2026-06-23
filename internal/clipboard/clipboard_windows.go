package clipboard

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	openClipboard  = user32.NewProc("OpenClipboard")
	closeClipboard = user32.NewProc("CloseClipboard")
	emptyClipboard = user32.NewProc("EmptyClipboard")
	setClipboard   = user32.NewProc("SetClipboardData")

	globalAlloc  = kernel32.NewProc("GlobalAlloc")
	globalLock   = kernel32.NewProc("GlobalLock")
	globalUnlock = kernel32.NewProc("GlobalUnlock")
	globalFree   = kernel32.NewProc("GlobalFree")
)

// CopyText copies text to the Windows clipboard using the native Win32 clipboard API.
func CopyText(text string) error {
	if r, _, err := openClipboard.Call(0); r == 0 {
		return fmt.Errorf("OpenClipboard failed: %w", windowsErr(err))
	}
	defer closeClipboard.Call()

	if r, _, err := emptyClipboard.Call(); r == 0 {
		return fmt.Errorf("EmptyClipboard failed: %w", windowsErr(err))
	}

	data := syscall.StringToUTF16(text)
	sizeBytes := uintptr(len(data) * 2)

	handle, _, err := globalAlloc.Call(gmemMoveable, sizeBytes)
	if handle == 0 {
		return fmt.Errorf("GlobalAlloc failed: %w", windowsErr(err))
	}

	locked, _, err := globalLock.Call(handle)
	if locked == 0 {
		globalFree.Call(handle)
		return fmt.Errorf("GlobalLock failed: %w", windowsErr(err))
	}

	dst := unsafe.Slice((*uint16)(unsafe.Pointer(locked)), len(data))
	copy(dst, data)
	globalUnlock.Call(handle)
	runtime.KeepAlive(data)

	if r, _, err := setClipboard.Call(cfUnicodeText, handle); r == 0 {
		globalFree.Call(handle)
		return fmt.Errorf("SetClipboardData failed: %w", windowsErr(err))
	}

	// On successful SetClipboardData, Windows owns the memory handle.
	return nil
}

func windowsErr(err error) error {
	if errno, ok := err.(syscall.Errno); ok && errno == 0 {
		return syscall.EINVAL
	}
	return err
}
