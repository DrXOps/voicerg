//go:build !windows

package wake

import (
	"context"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// VoskEngine — заглушка для не-Windows платформ.
type VoskEngine struct{}

// NewEngine создаёт VoskEngine (заглушка).
func NewEngine() *VoskEngine {
	return &VoskEngine{}
}

func (v *VoskEngine) Start(_ context.Context, modelPath, grammarJSON string) error {
	l := logger.Get()
	l.Warn("wake: Vosk недоступен на этой платформе (stub)")
	return nil
}

func (v *VoskEngine) FeedAudio(samples []int16, frameTS uint64) {}

func (v *VoskEngine) Events() <-chan WakeEvent { return nil }

func (v *VoskEngine) Stop() {}
