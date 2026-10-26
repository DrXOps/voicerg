package audio

import "math"

// VAD — простой детектор голосовой активности на основе RMS.
// Используется перед Vosk, чтобы не кормить распознаватель тишиной/шумом
// (снижает ложные срабатывания и нагрузку CPU).
type VAD struct {
	// RMS-порог: ниже считаем тишиной. 0..32768.
	threshold float64
	// minSilenceMs — паузы короче этой длительности не разрывают "активность",
	// чтобы не терять слово внутри фразы.
	onDelayFrames int
}

// NewVAD создаёт детектор. threshold — RMS, типично 300..800 для 16-бит.
func NewVAD(threshold float64) *VAD {
	if threshold <= 0 {
		threshold = 400
	}
	return &VAD{
		threshold: threshold,
	}
}

// RMS считает среднеквадратичную амплитуду фрейма.
func RMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		v := float64(s)
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(samples)))
}

// IsSpeech — true, если фрейм активен (RMS выше порога).
func (v *VAD) IsSpeech(samples []int16) bool {
	return RMS(samples) >= v.threshold
}
