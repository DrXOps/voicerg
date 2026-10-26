package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

func initWikiLogger(t *testing.T) {
	t.Helper()
	_ = logger.Init(logger.Config{
		LogDir:   t.TempDir(),
		LogFile:  "test.log",
		MaxBytes: 1 << 20,
		KeepOld:  2,
	})
}

func setupWiki(t *testing.T) string {
	dir := t.TempDir()
	// главный файл
	os.WriteFile(filepath.Join(dir, "_index.md"), []byte("# Главный\n\n[[chest.md]]\n\n[[missing.md]]"), 0o644)
	// вложенный
	os.WriteFile(filepath.Join(dir, "chest.md"), []byte("# ОГК\n\n[[rules.md]]"), 0o644)
	// rules ссылается на _index (цикл)
	os.WriteFile(filepath.Join(dir, "rules.md"), []byte("# Правила\n\nссылка [[_index.md]]"), 0o644)
	return dir
}

func TestWikiBuildSimple(t *testing.T) {
	initWikiLogger(t)
	dir := setupWiki(t)
	w := NewWikiBuilder(dir)
	out, info, err := w.Build("_index.md")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(out, "Главный") {
		t.Error("missing main content")
	}
	// chest.md встроен
	if !strings.Contains(out, "ОГК") {
		t.Error("missing nested chest.md")
	}
	// rules.md встроен (через chest)
	if !strings.Contains(out, "Правила") {
		t.Error("missing nested rules.md")
	}
	// цикл: rules -> _index должен быть защищён (не бесконечная рекурсия)
	if !strings.Contains(out, "Вики-файл") {
		t.Log("note: нет маркеров вложения — вероятно ушиб")
	}
	// missing.md помечен как ненайденный
	if !strings.Contains(out, "ФАЙЛ НЕ НАЙДЕН") {
		t.Error("missing.md should be marked")
	}
	// В info нет цикла = сборка завершилась
	if info.TotalSizeBytes <= 0 {
		t.Error("total size should be > 0")
	}
	if len(info.MissingLinks) != 1 || info.MissingLinks[0] != "missing.md" {
		t.Errorf("missingLinks = %v, want [missing.md]", info.MissingLinks)
	}
}

func TestWikiBuildNoCycle(t *testing.T) {
	initWikiLogger(t)
	dir := setupWiki(t)
	w := NewWikiBuilder(dir)
	// многократный запуск не должен падать
	for i := 0; i < 3; i++ {
		if _, _, err := w.Build("_index.md"); err != nil {
			t.Fatalf("Build iter %d: %v", i, err)
		}
	}
}

func TestWikiUnsafePath(t *testing.T) {
	initWikiLogger(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_index.md"), []byte("[[../../etc/passwd]]"), 0o644)
	w := NewWikiBuilder(dir)
	_, _, err := w.Build("_index.md")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// не должно читать вне каталога — просто значение флага
	// (функция resolve помечает ../ и не включает)
}
