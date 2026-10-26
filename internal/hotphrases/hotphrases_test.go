package hotphrases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

func initLog(t *testing.T) {
	t.Helper()
	_ = logger.Init(logger.Config{LogDir: t.TempDir(), LogFile: "t.log", MaxBytes: 1 << 20, KeepOld: 2})
}

func TestLoadAndApply(t *testing.T) {
	initLog(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "hotphrases.json")
	os.WriteFile(path, []byte(`{
		"стандартное сердце": "Тень сердца обычной конфигурации, размеры не увеличены.",
		"чистые лёгкие": "Легочный рисунок не изменен."
	}`), 0o644)

	s := New(path)
	if err := s.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	text, n := s.Apply("Например, стандартное сердце. Чистые лёгкие.")
	if n != 2 {
		t.Errorf("заменено %d, want 2", n)
	}
	if !strings.Contains(text, "Тень сердца") {
		t.Errorf("горячая фраза не заменена: %q", text)
	}
	if strings.Contains(text, "стандартное сердце") {
		t.Errorf("длинная фраза должна быть заменена: %q", text)
	}
}

func TestLoadMissingFile(t *testing.T) {
	initLog(t)
	s := New(filepath.Join(t.TempDir(), "none.json"))
	if err := s.Load(); err != nil {
		t.Errorf("отсутствующий файл не должен быть ошибкой: %v", err)
	}
}

func TestApplyEmpty(t *testing.T) {
	initLog(t)
	s := New("x")
	text, n := s.Apply("")
	if n != 0 || text != "" {
		t.Errorf("пустой вход: text=%q n=%d", text, n)
	}
	text, n = s.Apply("anything")
	if n != 0 || text != "anything" {
		t.Errorf("пустое хранилище: text=%q n=%d", text, n)
	}
}

func TestCaseInsensitive(t *testing.T) {
	initLog(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "hp.json")
	os.WriteFile(path, []byte(`{"Без Патологии": "Патологических изменений не выявлено."}`), 0o644)
	s := New(path)
	s.Load()
	text, _ := s.Apply("Аннотация. без патологии.")
	if !strings.Contains(text, "Патологических") {
		t.Errorf("регистронезависимость не работает: %q", text)
	}
}
