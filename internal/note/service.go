// Package note — голосовые заметки о недочётах (NoteRecording → NoteProcessing).
// Записывает структурированные заметки в notes/feedback.md для последующего
// анализа (например, скормить Claude Code для улучшения).
package note

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Note — структурированная заметка.
type Note struct {
	Category string
	Title    string
	Body     string // полный текст заметки (разобранный LLM)
}

// Service — запись заметок в notes/feedback.md.
type Service struct {
	notesFile string
	mu        chan struct{} // для последовательной записи (либо sync.Mutex)
}

// New создаёт сервис заметок. path — путь к notes/feedback.md (абсолютный или
// относительно cwd).
func New(notesFile string) *Service {
	return &Service{
		notesFile: notesFile,
		mu:        make(chan struct{}, 1), // буфер 1 — один пишущий
	}
}

// WriteNote добавляет заметку в notes/feedback.md.
// Если файла нет — создаёт с заголовком сессии.
// Если первая заметка за сессию — добавляет заголовок сессии.
// Каждая заметка — через разделитель `---`.
func (s *Service) WriteNote(n Note) error {
	s.mu <- struct{}{}
	defer func() { <-s.mu }()

	_ = os.MkdirAll(filepath.Dir(s.notesFile), 0o755)

	// Определяем, нужен ли заголовок сессии.
	isFirst := false
	if _, err := os.Stat(s.notesFile); err != nil {
		isFirst = true
	}

	f, err := os.OpenFile(s.notesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("note: open %s: %w", s.notesFile, err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	now := time.Now()

	// Если пустой файл — пишем заголовок сессии.
	if isFirst {
		fmt.Fprintf(w, "## Сессия: %s (Windows)\n\n", now.Format("2006-01-02 15:04"))
	}

	// Разделитель между заметками (кроме первой в файле).
	if !isFirst {
		fmt.Fprintln(w, "---")
	}
	fmt.Fprintf(w, "### %s — %s\n", fixCategory(n.Category), n.Title)

	// Поля из LLM-структуры. Ищем в n.Body ключевые строки.
	fields := parseFields(n.Body)

	fmt.Fprintf(w, "**Время:** %s\n", now.Format("15:04:05"))
	fmt.Fprintf(w, "**Категория:** %s\n", n.Category)
	if v, ok := fields["Что произошло"]; ok && v != "" {
		fmt.Fprintf(w, "**Что произошло:** %s\n", v)
	}
	if v, ok := fields["Ожидалось"]; ok && v != "" {
		fmt.Fprintf(w, "**Ожидалось:** %s\n", v)
	}
	if v, ok := fields["Контекст"]; ok && v != "" {
		fmt.Fprintf(w, "**Контекст:** %s\n", v)
	}
	if v, ok := fields["Возможное решение"]; ok && v != "" {
		fmt.Fprintf(w, "**Возможное решение:** %s\n", v)
	}
	if v, ok := fields["Цитата"]; ok && v != "" {
		fmt.Fprintf(w, "**Цитата:** > %s\n", v)
	}
	fmt.Fprintln(w) // пустая строка для читаемости

	if err := w.Flush(); err != nil {
		return fmt.Errorf("note: flush: %w", err)
	}

	return nil
}

// parseFields извлекает пары ключ: значение из текста вида:
// Категория: bug
// Заголовок: ...
func parseFields(body string) map[string]string {
	res := make(map[string]string)
	lines := strings.Split(body, "\n")
	var currentKey string
	var currentVal strings.Builder

	for _, ln := range lines {
		trimmed := ln
		idx := strings.Index(trimmed, ": ")
		if idx > 0 && idx < 40 {
			// сохраняем предыдущую
			if currentKey != "" {
				res[currentKey] = strings.TrimSpace(currentVal.String())
				currentVal.Reset()
			}
			currentKey = strings.TrimSpace(trimmed[:idx])
			currentVal.WriteString(strings.TrimSpace(trimmed[idx+2:]))
		} else if currentKey != "" {
			// продолжение значения
			if currentVal.Len() > 0 {
				currentVal.WriteString(" ")
			}
			currentVal.WriteString(strings.TrimSpace(trimmed))
		}
	}
	if currentKey != "" {
		res[currentKey] = strings.TrimSpace(currentVal.String())
	}
	return res
}

var validCategories = map[string]string{
	"bug":     "bug",
	"ux":      "ux",
	"prompt":  "prompt",
	"data":    "data",
	"idea":    "idea",
	"unclear": "unclear",
}

func fixCategory(cat string) string {
	if _, ok := validCategories[cat]; ok {
		return cat
	}
	return "unclear"
}
