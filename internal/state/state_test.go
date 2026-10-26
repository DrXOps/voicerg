package state

import (
	"testing"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

func initTestLogger(t *testing.T) {
	t.Helper()
	_ = logger.Init(logger.Config{
		LogDir:   t.TempDir(),
		LogFile:  "test.log",
		MaxBytes: 1 << 20,
		KeepOld:  2,
	})
}

func TestTransitions(t *testing.T) {
	initTestLogger(t)
	sm := New(nil)
	if sm.Current() != Idle {
		t.Fatalf("init state = %v, want Idle", sm.Current())
	}

	// Idle -> Recording разрешён
	if ok, reason := sm.Can([]State{Idle}, Recording, "поехали"); !ok {
		t.Fatalf("Idle->Recording refused: %s", reason)
	}
	if sm.Current() != Recording {
		t.Errorf("current = %v, want Recording", sm.Current())
	}

	// Idle -> Recording теперь НЕ разрешён (мы в Recording)
	if ok, _ := sm.Can([]State{Idle}, Recording, "x"); ok {
		t.Error("Idle->Recording should be refused while in Recording")
	}

	// Recording -> Done разрешён
	if ok, _ := sm.Can([]State{Recording}, Done, "заключение"); !ok {
		t.Error("Recording->Done should be allowed")
	}
	if sm.Current() != Done {
		t.Errorf("current = %v, want Done", sm.Current())
	}

	// Done -> Idle
	if ok, _ := sm.Can([]State{Done}, Idle, "reset"); !ok {
		t.Error("Done->Idle should be allowed")
	}
}

func TestHistIter(t *testing.T) {
	initTestLogger(t)
	sm := New(nil)
	sm.Can([]State{Idle}, Recording, "wake1")
	sm.Can([]State{Recording}, Done, "stop1")
	h := sm.History(10)
	if len(h) != 2 {
		t.Fatalf("history len = %d, want 2", len(h))
	}
	if h[1].To != "Done" {
		t.Errorf("last history To = %q", h[1].To)
	}
	// ограничение истории 64
	sm.Can([]State{Done}, Idle, "reset")
}

func TestDurationInCurrent(t *testing.T) {
	initTestLogger(t)
	sm := New(nil)
	if sm.DurationInCurrent() < 0 {
		t.Error("negative duration")
	}
}

func TestStatesNames(t *testing.T) {
	want := map[State]string{
		Idle: "Idle", Recording: "Recording", Processing: "Processing",
		Done: "Done", CommandRecording: "CommandRecording", IntentProcessing: "IntentProcessing",
		NoteRecording: "NoteRecording", NoteProcessing: "NoteProcessing",
	}
	for s, name := range want {
		if s.String() != name {
			t.Errorf("state %d = %q, want %q", int(s), s.String(), name)
		}
	}
}
