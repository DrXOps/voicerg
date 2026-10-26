package wake

import (
	"strings"

	"github.com/vasya/protocol-voice-assistant/internal/config"
)

// markerMatcher классифицирует фразу по маркерным словам и спискам.
type markerMatcher struct {
	wakeMarkers     []string
	stopMarkers     []string
	noteWakeWords   []string
	profileTriggers map[string]string
}

func newMarkerMatcher(cfg *config.Config) *markerMatcher {
	mm := &markerMatcher{
		wakeMarkers:     cfg.MarkerWords.WakeMarkers,
		stopMarkers:     cfg.MarkerWords.StopMarkers,
		noteWakeWords:   cfg.NoteWakeWords,
		profileTriggers: cfg.ProfileTriggers,
	}
	// копируем slice, т.к. config слайсы могут мутабелировать в других местах
	_ = mm
	return mm
}

// classify возвращает (eventType, profile, source):
//   - eventType: "" (нет совпадения), "wake", "stop", "note"
//   - profile: "knee"/"chest"/...
//   - source: "wake_words" / "marker" / "stop_words" / "note_wake"
func (m *markerMatcher) classify(phrase string) (eventType, profile, source string) {
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	if phrase == "" {
		return "", "", ""
	}

	// 1. note_wake_words (приоритетно — заметки)
	for _, nw := range m.noteWakeWords {
		if strings.Contains(phrase, nw) {
			return "note", "", "note_wake"
		}
	}

	// 2. Есть stop-маркер или stop_word?
	for _, sm := range m.stopMarkers {
		if strings.Contains(phrase, sm) {
			return "stop", "", "stop_words"
		}
	}

	// 3. Есть wake-маркер?
	hasWake := false
	for _, wm := range m.wakeMarkers {
		if strings.Contains(phrase, wm) {
			hasWake = true
			break
		}
	}
	if !hasWake {
		return "", "", ""
	}

	// 4. Есть профиль?
	for word, file := range m.profileTriggers {
		if strings.Contains(phrase, word) {
			return "wake", strings.TrimSuffix(file, ".md"), "marker"
		}
	}

	// 5. Только wake-маркер без профиля → intent (CommandRecording)
	return "intent", "", "marker"
}

func containsAny(phrase string, words []string) bool {
	for _, w := range words {
		if strings.Contains(phrase, w) {
			return true
		}
	}
	return false
}
