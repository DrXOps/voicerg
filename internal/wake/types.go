package wake

import "context"

// WakeEvent — событие, эмитируемое WakeEngine.
type WakeEvent struct {
	// Тип события: wake | stop | note | intent (когда только "вася").
	Type string
	// Распознанная фраза (как её увидел Vosk).
	Phrase string
	// Профиль исследования (пусто, если не определён).
	Profile string
	// Confidence Vosk (0.0-1.0, если доступно).
	Confidence float64
	// Source — какой триггер сработал: "wake_words" | "marker" | "stop_words" | "note_wake"
	Source string
	// AudioFrameTS — временная метка фрейма, с которого началась фраза
	AudioFrameTS uint64
	// DetectionLatencyMs — задержка детекции
	DetectionLatencyMs int64
}

// Engine — интерфейс детекции wake/stop-фраз.
type Engine interface {
	// Start запускает движок. Из вызывающей стороны — читать Events().
	Start(ctx context.Context, modelPath string, grammarJSON string) error
	// FeedAudio передаёт очередной чанк S16LE моно.
	FeedAudio(samples []int16, frameTS uint64)
	// Events возвращает канал событий.
	Events() <-chan WakeEvent
	// Stop завершает работу движка.
	Stop()
}
