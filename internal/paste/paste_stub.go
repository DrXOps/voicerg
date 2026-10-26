//go:build !windows

package paste

import (
	"errors"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// ErrNotSupported — безопасная заглушка на не-Windows платформах.
var ErrNotSupported = errors.New("paste: not supported on this platform")

func activeWindow() Window {
	return Window{HWND: 0, Valid: false}
}

func pasteViaKeys() error {
	logger.Get().Warn("paste: pasteViaKeys недоступен на этой платформе")
	return ErrNotSupported
}

func setClipboard(text string) error {
	// На linux можно использовать pbcopy/xsel, но для тестов — просто локальный
	// кэш, чтобы логика «в буфер» работала.
	if err := setClipboardLocal(text); err != nil {
		return err
	}
	logger.Get().Debug("paste: (linux) текст в локальном буфере")
	return nil
}

func getClipboard() (string, error) {
	return getClipboardLocal()
}
