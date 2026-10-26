package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultGeneratesConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// При отсутствии config.json должен создаться файл-образец.
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatalf("config.json not written: %v", err)
	}
	if cfg.WorkingDir() != dir {
		t.Errorf("workingDir = %q, want %q", cfg.WorkingDir(), dir)
	}
	if len(cfg.WakeWords) == 0 {
		t.Error("wake_words empty")
	}
	if len(cfg.ProfileTriggers) == 0 {
		t.Error("profile_triggers empty")
	}
	// конфиг по умолчанию содержит "вася"
	found := false
	for _, w := range cfg.WakeWords {
		if w == "вася" {
			found = true
		}
	}
	if !found {
		t.Error(`"вася" not in wake_words`)
	}
}

func TestLoadExistingLowercases(t *testing.T) {
	dir := t.TempDir()
	sample := `{
		"wake_words": ["ВасЯ ПиШем"],
		"stop_words": ["ЗАКЛЮЧЕНИЕ"],
		"marker_words": {"wake_markers": ["ПоЕхАли"]}
	}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WakeWords[0] != "вася пишем" {
		t.Errorf("wake normalized = %q", cfg.WakeWords[0])
	}
	if cfg.StopWords[0] != "заключение" {
		t.Errorf("stop normalized = %q", cfg.StopWords[0])
	}
	if cfg.MarkerWords.WakeMarkers[0] != "поехали" {
		t.Errorf("marker normalized = %q", cfg.MarkerWords.WakeMarkers[0])
	}
	if cfg.Hash() == "" {
		t.Error("hash empty")
	}
}

func TestResolve(t *testing.T) {
	dir := "/tmp/vasya-test"
	c := &Config{workingDir: dir}
	if got := c.Resolve("prompts"); got != "/tmp/vasya-test/prompts" {
		t.Errorf("Resolve rel = %q", got)
	}
	if got := c.Resolve("/abs/x"); got != "/abs/x" {
		t.Errorf("Resolve abs = %q", got)
	}
}
