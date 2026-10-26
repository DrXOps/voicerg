//go:build windows

package paste

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Windows User32 API (упрощённый доступ через syscall).
var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procSendInput                = user32.NewProc("SendInput")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

// keybd — тип для SendInput (KEYBDINPUT).
type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type input struct {
	typ uint32
	ki  keybdInput
	_   [8]byte // padding для объединения (MOUSEINPUT/KEYBDINPUT/HARDWAREINPUT)
}

const (
	inputKeyboard = 1
	keyEventKeyUp = 0x0002
	keyEventScan  = 0x0008
	// VK codes
	vkControl = 0x11
	vkV       = 0x56
)

func activeWindow() Window {
	hwnd, _, _ := procGetForegroundWindow.Call()
	w := Window{HWND: hwnd}
	if hwnd != 0 {
		w.Valid = true
		w.Title = getWindowTitle(hwnd)
		w.Process = processNameFromWindow(hwnd)
	}
	return w
}

func getWindowTitle(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return "?"
	}
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
	return syscall.UTF16ToString(buf)
}

func processNameFromWindow(hwnd uintptr) string {
	// Получить PID окна, затем имя процесса (упрощённо: вернём hex).
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return fmt.Sprintf("pid=%d", pid)
}

// pasteViaKeys устанавливает фокус (SetForegroundWindow) и шлёт Ctrl+V.
func pasteViaKeys() error {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd != 0 {
		procSetForegroundWindow.Call(hwnd)
	}

	ctrlDown := newInput().key(vkControl, 0)   // key down
	vDown := newInput().key(vkV, keyEventScan) // v down (с scancode)
	vUp := newInput().keyUp(vkV, keyEventScan) // v up
	ctrlUp := newInput().keyUp(vkControl, 0)   // ctrl up

	inputs := []input{ctrlDown, vDown, vUp, ctrlUp}

	ret, _, _ := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(unsafe.Sizeof(input{})),
	)
	if int(ret) != len(inputs) {
		return fmt.Errorf("SendInput вернул %d из %d", int(ret), len(inputs))
	}
	return nil
}

// --- вспомогательные построители input ---

type inputBuilder struct {
	in input
}

func newInput() *inputBuilder {
	return &inputBuilder{}
}

func (b *inputBuilder) key(vk uint16, flags uint32) input {
	b.in.typ = inputKeyboard
	b.in.ki.wVk = vk
	b.in.ki.dwFlags = flags
	return b.in
}

func (b *inputBuilder) keyUp(vk uint16, flags uint32) input {
	b.in.typ = inputKeyboard
	b.in.ki.wVk = vk
	b.in.ki.dwFlags = flags | keyEventKeyUp
	return b.in
}
