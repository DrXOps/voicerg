package audio

import "context"

// SampleRate — частота дискретизации (Гц) для всего конвейера.
const SampleRate = 16000

// DeviceInfo описывает найденное устройство.
type DeviceInfo struct {
	ID   string
	Name string
}

// Chunk — блок отсчётов S16LE моно.
type Chunk struct {
	Samples []int16
	// FrameTS — временная метка фрейма начала (счётчик сэмплов) для хронометража.
	FrameTS uint64
}

// Source — интерфейс источника аудио.
type Source interface {
	// Start запускает захват; device — подстрока имени (пусто = системное).
	Start(ctx context.Context, device string) error
	// Events возвращает канал с аудио-чанками.
	Events() <-chan Chunk
	// Stop завершает захват.
	Stop()
	// ListDevices перечисляет доступные устройства захвата.
	ListDevices() ([]DeviceInfo, error)
}
