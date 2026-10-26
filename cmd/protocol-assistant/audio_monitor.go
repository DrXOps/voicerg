package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/audio"
	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// monitorAudio читает канал чанков, раз в 10 секунд пишет агрегированную
// статистику audio_chunk_stats в events.jsonl. На Linux (заглушка) канал
// пустой и минутник просто не выводит записей.
func monitorAudio(src audio.Source, lg *logger.Logger) {
	ch := src.Events()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var (
		count        int
		rmsSum       float64
		rmsMax       float64
		silenceCount int
		vad          = audio.NewVAD(0)
	)

	flush := func() {
		if count == 0 {
			return
		}
		rmsAvg := rmsSum / float64(count)
		lg.Event("audio_chunk_stats", map[string]any{
			"rms_avg":         round2(rmsAvg),
			"rms_max":         round2(rmsMax),
			"silence_ratio":   round4(float64(silenceCount) / float64(count)),
			"chunks_received": count,
			"chunks_dropped":  0,
		})
		count = 0
		rmsSum = 0
		rmsMax = 0
		silenceCount = 0
	}

	for {
		select {
		case ck, ok := <-ch:
			if !ok {
				flush()
				return
			}
			count++
			r := audio.RMS(ck.Samples)
			rmsSum += r
			if r > rmsMax {
				rmsMax = r
			}
			if !vad.IsSpeech(ck.Samples) {
				silenceCount++
			}
		case <-ticker.C:
			flush()
		}
	}
}

// writeSessionSummary — фиксатор summary (заглушка, наполнится в этапах 6-12).
// Сохраняется рядом с exe в logs/sessions/.
func writeSessionSummary(lg *logger.Logger, dir string) {
	sessionFile := filepath.Join(dir, "logs", "sessions",
		time.Now().Format("2006-01-02_150405")+"_summary.json")
	_ = os.MkdirAll(filepath.Dir(sessionFile), 0o755)
	summary := map[string]any{
		"session_id": lg.SessionID(),
		"started":    lg.StartedAt().Format(time.RFC3339),
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	_ = os.WriteFile(sessionFile, b, 0o644)
	lg.Info("session_summary записан: " + sessionFile)
}

func round2(f float64) float64 { return float64(int(f*100)) / 100 }
func round4(f float64) float64 { return float64(int(f*10000)) / 10000 }
