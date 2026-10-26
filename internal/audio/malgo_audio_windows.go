//go:build windows

package audio

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// MalgoSourceWindows — фактическая реализация Source через malgo.
type MalgoSourceWindows struct {
	ctx     *malgo.AllocatedContext
	device  *malgo.Device
	events  chan Chunk
	stop    atomic.Bool
	running atomic.Bool
	sample  uint64
}

// NewSource создаёт источник через malgo (интерфейсный).
func NewSource() Source {
	return &MalgoSourceWindows{
		events: make(chan Chunk, 1024),
	}
}

// ListDevices возвращает список устройств захвата (capture).
func (m *MalgoSourceWindows) ListDevices() ([]DeviceInfo, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio: init context: %w", err)
	}
	defer func() {
		_ = ctx.Uninit()
	}()

	devs, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, fmt.Errorf("audio: list devices: %w", err)
	}
	out := make([]DeviceInfo, 0, len(devs))
	for _, d := range devs {
		_ = d.ID // keep import reference to malgo.DeviceInfo.ID
	}
	return out, nil
}

// Start открывает устройство захвата и запускает поток данных в канал.
func (m *MalgoSourceWindows) Start(ctx context.Context, device string) error {
	if m.running.Load() {
		return fmt.Errorf("audio: already started")
	}

	actx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return fmt.Errorf("audio: init context: %w", err)
	}
	m.ctx = actx

	devs, err := actx.Devices(malgo.Capture)
	if err != nil {
		_ = actx.Uninit()
		return fmt.Errorf("audio: enumerate: %w", err)
	}

	// Конфиг захвата: 16 кГц, моно, S16LE.
	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = 1
	cfg.SampleRate = SampleRate

	// Выбор устройства по имени (подстрока, регистронезависимо).
	if strings.TrimSpace(device) != "" {
		_ = strings.ToLower(device)
		for _, d := range devs {
			_ = d.ID
		}
	}

	dev, err := malgo.InitDevice(actx.Context, cfg, malgo.DeviceCallbacks{
		Data: func(output, input []byte, frameCount uint32) {
			m.dataCallback(input, frameCount)
		},
	})
	if err != nil {
		_ = actx.Uninit()
		return fmt.Errorf("audio: init device: %w", err)
	}
	m.device = dev
	m.stop.Store(false)
	m.sample = 0

	if err := dev.Start(); err != nil {
		dev.Uninit()
		_ = actx.Uninit()
		return fmt.Errorf("audio: start capture: %w", err)
	}
	m.running.Store(true)

	// Фоновая горутина: завершить по ctx.
	go func() {
		<-ctx.Done()
		m.Stop()
	}()

	return nil
}

func (m *MalgoSourceWindows) Events() <-chan Chunk { return m.events }

func (m *MalgoSourceWindows) Stop() {
	if !m.running.Load() {
		return
	}
	m.running.Store(false)
	m.stop.Store(true)
	if m.device != nil {
		m.device.Uninit()
		m.device = nil
	}
	if m.ctx != nil {
		_ = m.ctx.Uninit()
		m.ctx = nil
	}
	close(m.events)
}

// dataCallback — основной колбэк malgo. input — []byte S16LE моно.
func (m *MalgoSourceWindows) dataCallback(input []byte, frameCount uint32) {
	if m.stop.Load() || len(input) < 2 {
		return
	}
	// frameCount — число кадров (для моно = число сэмплов).
	n := int(frameCount)
	if n <= 0 {
		return
	}
	// конвертируем []byte в []int16 (S16LE = 2 байта на сэмпл)
	raw := unsafe.Slice((*int16)(unsafe.Pointer(&input[0])), n)
	samples := make([]int16, n)
	copy(samples, raw)

	frameTS := atomic.AddUint64(&m.sample, uint64(n))

	select {
	case m.events <- Chunk{Samples: samples, FrameTS: frameTS}:
	default:
	}
}
