//go:build windows

package wake

import (
	"context"
	"fmt"
	"sync"
	"unicode/utf8"

	vosk "github.com/alphacep/vosk-api/go"

	"github.com/vasya/protocol-voice-assistant/internal/config"
	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// VoskEngine реализует графический детектор wake/stop-фраз через Vosk Grammar Mode.
type VoskEngine struct {
	model      *vosk.VoskModel
	rec        *vosk.VoskRecognizer
	events     chan WakeEvent
	mu         sync.Mutex
	started    bool
	lastResult string
	matcher    *markerMatcher
}

// NewEngine создаёт VoskEngine.
func NewEngine() *VoskEngine {
	return &VoskEngine{
		events: make(chan WakeEvent, 32),
	}
}

// Start загружает модель и создаёт распознаватель с грамматикой.
func (v *VoskEngine) Start(ctx context.Context, modelPath, grammarJSON string) error {
	l := logger.Get()
	if v.started {
		return fmt.Errorf("wake: уже запущен")
	}

	l.Info(fmt.Sprintf("wake: загружаю модель из %s", modelPath))
	model, err := vosk.NewModel(modelPath)
	if err != nil {
		return fmt.Errorf("wake: NewModel(%s): %w", modelPath, err)
	}
	v.model = model

	l.Info("wake: создаю RecognizerGrm...")
	rec, err := vosk.NewRecognizerGrm(model, 16000, grammarJSON)
	if err != nil {
		return fmt.Errorf("wake: NewRecognizerGrm: %w", err)
	}
	v.rec = rec
	v.started = true

	l.Info("wake: Vosk Grammar Mode активен, грамматика загружена")
	return nil
}

// FeedAudio подаёт входные сэмплы S16LE моно в Vosk.
func (v *VoskEngine) FeedAudio(samples []int16, frameTS uint64) {
	if !v.started || v.rec == nil {
		return
	}
	raw := int16SliceToBytes(samples)
	v.rec.AcceptWaveform(raw)

	resultJSON := v.rec.Result()
	v.processResult(resultJSON, frameTS)
}

func (v *VoskEngine) processResult(jsonStr string, frameTS uint64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if jsonStr == "" || jsonStr == "{}" || jsonStr == v.lastResult {
		return
	}
	v.lastResult = jsonStr

	phrase := extractPhrase(jsonStr)
	if phrase == "" {
		return
	}

	l := logger.Get()
	evType, profile, source := v.matcher.classify(phrase)

	if evType == "" {
		return
	}

	ev := WakeEvent{
		Type:               evType,
		Phrase:             phrase,
		Profile:            profile,
		Confidence:         0.0,
		Source:             source,
		AudioFrameTS:       frameTS,
		DetectionLatencyMs: 0,
	}

	select {
	case v.events <- ev:
	default:
		l.Warn("wake: канал событий переполнен, дроп события")
	}
}

// Events возвращает канал wake-событий.
func (v *VoskEngine) Events() <-chan WakeEvent { return v.events }

// Stop освобождает ресурсы.
func (v *VoskEngine) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.started {
		return
	}
	v.started = false
	if v.rec != nil {
		v.rec.Free()
		v.rec = nil
	}
	if v.model != nil {
		v.model.Free()
		v.model = nil
	}
	close(v.events)
}

// SetConfig передаёт списки фраз для маркер-матчинга.
func (v *VoskEngine) SetConfig(cfg *config.Config) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.matcher = newMarkerMatcher(cfg)
}

// int16SliceToBytes конвертирует []int16 в []byte (little-endian).
func int16SliceToBytes(samples []int16) []byte {
	b := make([]byte, len(samples)*2)
	for i, s := range samples {
		b[i*2] = byte(s)
		b[i*2+1] = byte(s >> 8)
	}
	return b
}

// Ensure import is used.
var _ = utf8.ValidString
