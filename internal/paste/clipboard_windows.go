//go:build windows

package paste

import (
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	cfUnicodText = 13 // CF_UNICODETEXT
)

var (
	user32Cb             = syscall.NewLazyDLL("user32.dll")
	kernel32Cb           = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard    = user32Cb.NewProc("OpenClipboard")
	procEmptyClipboard   = user32Cb.NewProc("EmptyClipboard")
	procSetClipboardData = user32Cb.NewProc("SetClipboardData")
	procCloseClipboard   = user32Cb.NewProc("CloseClipboard")
	procGetClipboardData = user32Cb.NewProc("GetClipboardData")
	procGlobalAlloc      = kernel32Cb.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32Cb.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32Cb.NewProc("GlobalUnlock")
)

const (
	gmemMoveable = 0x0002
	gmemZeroInit = 0x0040
)

// setClipboard кладёт текстовую строку в буфер обмена Windows.
func setClipboard(text string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	utf := utf16.Encode([]rune(text + "\x00"))
	// резервируем глобальную память
	bytesSize := len(utf) * 2
	hMem, _, _ := procGlobalAlloc.Call(uintptr(gmemMoveable|gmemZeroInit), uintptr(bytesSize))
	if hMem == 0 {
		return syscall.GetLastError()
	}
	ptr, _, e1 := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return e1
	}
	// копируем UTF-16
	dest := (*[1 << 20]uint16)(unsafe.Pointer(ptr))[:len(utf):len(utf)]
	copy(dest, utf)
	procGlobalUnlock.Call(hMem)

	// открываем буфер и записываем
	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return syscall.GetLastError()
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	if r, _, _ := procSetClipboardData.Call(cfUnicodText, hMem); r == 0 {
		return syscall.GetLastError()
	}
	// после успешного SetClipboardData владение hMem переходит к системе
	return nil
}

// getClipboard читает текст из буфера обмена Windows.
func getClipboard() (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return "", syscall.GetLastError()
	}
	defer procCloseClipboard.Call()

	hMem, _, _ := procGetClipboardData.Call(cfUnicodText)
	if hMem == 0 {
		return "", nil // буфер пуст/нет текста
	}
	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", syscall.GetLastError()
	}
	defer procGlobalUnlock.Call(hMem)

	// читаем до NULL-терминатора (максимум 1<<20)
	runes := make([]uint16, 0)
	p := unsafe.Pointer(ptr)
	pu16 := (*uint16)(p)
	for i := 0; i < 1<<20; i++ {
		c := *((*uint16)(unsafe.Pointer(uintptr(p) + uintptr(i*2))))
		if c == 0 {
			break
		}
		runes = append(runes, c)
	}
	_ = pu16
	return string(utf16.Decode(runes)), nil
}
