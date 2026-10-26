package protocol

import "strings"

// BlockParser извлекает блоки протокола по маркерам в начале строк.
// Поддерживаемые маркеры соответствуют format из prompts/rules.md.
type BlockParser struct {
	text string
}

// NewBlockParser создаёт парсер.
func NewBlockParser(text string) *BlockParser {
	return &BlockParser{text: text}
}

// BlockNames — все блоки, которые IntenтHandler может вставить.
var BlockNames = []string{
	"Название",
	"ОПИСАНИЕ",
	"ЗАКЛЮЧЕНИЕ",
	"КЛИНИЧЕСКИЕ ДАННЫЕ",
	"ВОЗМОЖНОЕ ЗАКЛЮЧЕНИЕ",
	"РЕКОМЕНДАЦИИ",
}

// GetBlock возвращает текст блока по имени (как в маркере).
// Возвращает пустую строку, если блок не найден.
func (bp *BlockParser) GetBlock(name string) string {
	lines := strings.Split(bp.text, "\n")
	var out []string
	inBlock := false

	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		// маркер блока — имя в [ ] в начале строки
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			marker := strings.Trim(trimmed, "[]")
			// регистронезависимое сравнение
			if strings.EqualFold(marker, name) {
				inBlock = true
				out = append(out, ln)
				continue
			}
			if inBlock {
				break // следующий маркер — выход из текущего блока
			}
		}
		if inBlock {
			out = append(out, ln)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// FullText возвращает весь протокол.
func (bp *BlockParser) FullText() string { return bp.text }

// HotPhrases — хранилище подстановок (используется в генераторе).
type HotPhrases struct {
	// встроенные + загруженные из hotphrases.json (этап 8)
	Replacements map[string]string
}
