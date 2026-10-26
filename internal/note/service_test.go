package note

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteNoteCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "feedback.md")
	s := New(path)

	n := Note{
		Category: "bug",
		Title:    "не сработала стоп-фраза",
		Body:     "Категория: bug\nЧто произошло: диктовка не остановилась\nОжидалось: переход в обработку\nКонтекст: Recording\nВозможное решение: проверить грамматику",
	}
	if err := s.WriteNote(n); err != nil {
		t.Fatalf("WriteNote: %v", err)
	}

	data, _ := os.ReadFile(path)
	text := string(data)

	// Заголовок сессии
	if !strings.Contains(text, "## Сессия:") {
		t.Error("нет заголовка сессии")
	}
	// Категория
	if !strings.Contains(text, "### bug —") {
		t.Errorf("нет заголовка заметки: %s", text)
	}
	// Поля
	if !strings.Contains(text, "**Что произошло:** диктовка не остановилась") {
		t.Errorf("нет поля Что произошло: %s", text)
	}
	if !strings.Contains(text, "**Ожидалось:** переход в обработку") {
		t.Errorf("нет поля Ожидалось: %s", text)
	}
}

func TestWriteMultipleNotes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "feedback.md")
	s := New(path)

	s.WriteNote(Note{Category: "idea", Title: "идея первая", Body: "Категория: idea\nЧто произошло: t1"})
	s.WriteNote(Note{Category: "ux", Title: "идея вторая", Body: "Категория: ux\nЧто произошло: t2"})

	data, _ := os.ReadFile(path)
	text := string(data)
	// Один заголовок сессии
	if strings.Count(text, "## Сессия:") != 1 {
		t.Errorf("заголовков сессии = %d, want 1", strings.Count(text, "## Сессия:"))
	}
	// Два разделителя нет (первая без '---'), но вторая заметка имеет
	if !strings.Contains(text, "---") {
		t.Error("нет разделителя между заметками")
	}
	if !strings.Contains(text, "### ux — идея вторая") {
		t.Error("нет второй заметки")
	}
}

func TestFixCategoryFallback(t *testing.T) {
	if got := fixCategory("bug"); got != "bug" {
		t.Errorf("fixCategory(bug) = %q", got)
	}
	if got := fixCategory("хз"); got != "unclear" {
		t.Errorf("fixCategory(хз) = %q, want unclear", got)
	}
}

func TestParseFields(t *testing.T) {
	body := "Что произошло: одно\nОжидалось: другое\nПервое: вторая строка"
	m := parseFields(body)
	if m["Что произошло"] != "одно" {
		t.Errorf("Что произошло = %q", m["Что произошло"])
	}
	if m["Ожидалось"] != "другое" {
		t.Errorf("Ожидалось = %q", m["Ожидалось"])
	}
}
