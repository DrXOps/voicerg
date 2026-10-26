package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerWritesBothFiles(t *testing.T) {
	resetForTest()
	dir := t.TempDir()
	lg := Init(Config{LogDir: dir, LogFile: "app.log", MaxBytes: 1 << 20, KeepOld: 3})
	lg.Info("hello world")
	lg.Event("state_change", map[string]any{"from": "Idle", "to": "Recording"})

	// human log
	hb, _ := os.ReadFile(filepath.Join(dir, "app.log"))
	if !strings.Contains(string(hb), "hello world") {
		t.Errorf("human log missing message: %s", hb)
	}
	if !strings.Contains(string(hb), "[INFO]") {
		t.Errorf("missing level: %s", hb)
	}

	// events
	eb, _ := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(eb)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 event line, got %d: %q", len(lines), eb)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("events.jsonl not valid json: %v", err)
	}
	if rec["event"] != "state_change" {
		t.Errorf("event name = %v", rec["event"])
	}
	if _, ok := rec["ts"]; !ok {
		t.Error("missing ts")
	}
	if rec["data"].(map[string]any)["to"] != "Recording" {
		t.Errorf("data.to = %v", rec["data"])
	}
}

func TestRotation(t *testing.T) {
	resetForTest()
	dir := t.TempDir()
	lg := Init(Config{LogDir: dir, LogFile: "app.log", MaxBytes: 500, KeepOld: 2})
	// пишем достаточно, чтобы превысить MaxBytes
	for i := 0; i < 100; i++ {
		lg.Info("012345678901234567890123456789012345678901234567890123456789")
	}
	resetForTest()
	// должна появиться ротация app.log.1
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), "app.log.1") {
			found = true
		}
	}
	if !found {
		t.Errorf("rotation backup not created, dir=%v", dir)
	}
	if lg.SessionID() == "" {
		t.Error("empty session id")
	}
}
