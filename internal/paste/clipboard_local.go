package paste

import "sync"

// Локальный буфер обмена для платформ без клипборд-системы (тестовая заглушка).
// Используется и на Linux-сборке, и в юнит-тестах windows-логики.
var (
	clipMu    sync.RWMutex
	clipValue string
)

func setClipboardLocal(text string) error {
	clipMu.Lock()
	defer clipMu.Unlock()
	clipValue = text
	return nil
}

func getClipboardLocal() (string, error) {
	clipMu.RLock()
	defer clipMu.RUnlock()
	return clipValue, nil
}
