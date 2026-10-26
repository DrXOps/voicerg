package wake

import "strings"

// extractPhrase извлекает поле "text" из JSON-ответа Vosk.
// Vosk возвращает {"text":"...", "partial":"...", ...}.
// Возвращает пустую строку, если не найдено.
func extractPhrase(jsonStr string) string {
	// простой поиск ключа "text" без полного парсинга (быстрее)
	jsonStr = strings.TrimSpace(jsonStr)

	// ищем "text": "
	key := `"text":"`
	idx := strings.Index(jsonStr, key)
	if idx < 0 {
		return ""
	}
	idx += len(key)
	end := strings.IndexByte(jsonStr[idx:], '"')
	if end < 0 {
		return ""
	}
	return jsonStr[idx : idx+end]
}
