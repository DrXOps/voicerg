//go:build !windows

package audio

import (
	"context"
	"errors"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// ErrNotSupported — захват недоступен на этой платформе.
var ErrNotSupported = errors.New("audio: capture not supported on this platform")

// NewSource создаёт источник аудио (заглушка).
func NewSource() Source {
	return &stubSource{}
}

type stubSource struct{}

func (m *stubSource) Start(_ context.Context, device string) error {
	l := logger.Get()
	l.Warn("audio: malgo source недоступен на этой платформе (capture stub)")
	return ErrNotSupported
}

func (m *stubSource) Events() <-chan Chunk { return nil }

func (m *stubSource) Stop() {}

func (m *stubSource) ListDevices() ([]DeviceInfo, error) {
	return nil, ErrNotSupported
}
