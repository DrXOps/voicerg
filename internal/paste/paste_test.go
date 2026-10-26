package paste

import (
	"testing"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

func initLog(t *testing.T) {
	t.Helper()
	_ = logger.Init(logger.Config{LogDir: t.TempDir(), LogFile: "t.log", MaxBytes: 1 << 20, KeepOld: 2})
}

func TestSmartPasteBuffersOnInvalidTarget(t *testing.T) {
	initLog(t)
	// Целевое окно не запомнено (invalid) — не вставляем, но кладём в буфер.
	target := Window{HWND: 0, Title: "no", Valid: false}
	res := SmartPaste(target, "важный текст", true)
	if res.Success != true {
		t.Errorf("Success = %v, want true (текст должен попасть в буфер)", res.Success)
	}
	if res.Pasted {
		t.Error("Pasted должно быть false при невалидном окне")
	}
	if got, _ := GetTextFromClipboard(); got != "важный текст" {
		t.Errorf("буфер не заполнен: %q", got)
	}
}

func TestSmartPasteWithPasteDisabled(t *testing.T) {
	initLog(t)
	target := Window{HWND: 0, Valid: false}
	res := SmartPaste(target, "текст", false) // pasteInActive=false
	if res.Success != true {
		t.Errorf("Success = %v", res.Success)
	}
	if res.Pasted {
		t.Error("Pasted должен быть false при отключённой вставке")
	}
	if got, _ := GetTextFromClipboard(); got != "текст" {
		t.Errorf("буфер = %q", got)
	}
}

func TestSmartPasteWindowMismatch(t *testing.T) {
	initLog(t)
	// Невалидное окно + вставка включена → fallback в буфер, не Пасте́д.
	target := Window{HWND: 123, Title: "запомненное", Valid: false}
	res := SmartPaste(target, "xyz", true)
	if res.Pasted {
		t.Error("Pasted должно быть false (промах окна на stub-платформе)")
	}
	if res.Error == "" {
		t.Error("должна быть причина отказа")
	}
}
