// Package paste — умная вставка текста в активное окно (Ctrl+V через SendInput),
// а также работа с буфером обмена. Windows-реализация через User32 (syscall);
// на других платформах — безопасная заглушка (только буфер).
package paste

import (
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// Window — дескриптор активного окна, запомненного до диктовки.
type Window struct {
	// HWND — native handle (Windows). На других платформах — 0.
	HWND uintptr
	// Title — заголовок окна на момент захвата.
	Title string
	// Process — имя процесса окна.
	Process string
	// Valid — окно захвачено успешно.
	Valid bool
}

// ActiveWindow возвращает текущее активное окно (платформозависимо).
func ActiveWindow() Window { return activeWindow() }

// SetTextInClipboard кладёт текст в системный буфер обмена.
func SetTextInClipboard(text string) error { return setClipboard(text) }

// GetTextFromClipboard читает текст из буфера.
func GetTextFromClipboard() (string, error) { return getClipboard() }

// PasteResult — результат попытки умной вставки.
type PasteResult struct {
	Success     bool
	Pasted      bool  // true = вставили; false = только в буфер
	Latency     int64 // мс
	Error       string
	WindowTitle string
}

// SmartPaste возвращает фокус в окно target и эмулирует Ctrl+V.
// Если текущее активное окно отличается — НЕ вставляем, помещаем в буфер.
func SmartPaste(target Window, text string, pasteInActive bool) PasteResult {
	l := logger.Get()
	start := time.Now()

	// Всегда кладём в буфер (fallback).
	if err := SetTextInClipboard(text); err != nil {
		l.Warn("paste: не удалось положить текст в буфер: " + err.Error())
		res := PasteResult{Pasted: false, Latency: time.Since(start).Milliseconds(), Error: "clipboard: " + err.Error()}
		return res
	}

	res := PasteResult{
		Success:     true,
		WindowTitle: target.Title,
	}

	// Если вставка в активное окно отключена — только буфер.
	if !pasteInActive {
		res.ApplyLatency(start)
		l.Info("paste: вставка отключена, текст в буфере")
		return res
	}

	// Защита от промаха: проверяем, что активное окно то же.
	if !target.Valid {
		res.ApplyLatency(start)
		res.Error = "целевое окно не запомнено"
		l.Warn("paste: " + res.Error)
		return res
	}

	cur := ActiveWindow()
	if cur.HWND != target.HWND {
		res.ApplyLatency(start)
		res.Error = "активное окно изменилось, НЕ вставляю (см. буфер)"
		l.Info("paste: " + res.Error)
		return res
	}

	// Возвращаем фокус (SetForegroundWindow) и вставляем Ctrl+V.
	if err := pasteViaKeys(); err != nil {
		res.ApplyLatency(start)
		res.Error = err.Error()
		l.Warn("paste: " + err.Error())
		return res
	}

	res.Pasted = true
	res.ApplyLatency(start)
	l.Info("paste: вставлено в «" + target.Title + "»")
	return res
}

func (r *PasteResult) ApplyLatency(start time.Time) {
	r.Latency = time.Since(start).Milliseconds()
}
