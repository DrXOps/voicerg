package stt

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEncodeWAVHeader(t *testing.T) {
	samples := []int16{100, 200, 300, 0, -100, -200}
	wav, err := encodeWAV(samples)
	if err != nil {
		t.Fatalf("encodeWAV: %v", err)
	}
	if len(wav) < 44 {
		t.Fatalf("wav too short: %d", len(wav))
	}

	// Проверяем RIFF-заголовок
	if !bytes.HasPrefix(wav, []byte("RIFF")) {
		t.Error("missing RIFF header")
	}
	if !bytes.Contains(wav, []byte("WAVE")) {
		t.Error("missing WAVE marker")
	}

	// Проверяем параметры: моно, 16 кГц, 16 бит
	fmtVal := wav[20:22]
	numCh := binary.LittleEndian.Uint16(fmtVal)
	if numCh != 1 {
		t.Errorf("channels = %d, want 1", numCh)
	}
	srVal := wav[24:28]
	sr := binary.LittleEndian.Uint32(srVal)
	if sr != 16000 {
		t.Errorf("sampleRate = %d, want 16000", sr)
	}
	bpsVal := wav[34:36]
	bps := binary.LittleEndian.Uint16(bpsVal)
	if bps != 16 {
		t.Errorf("bitsPerSample = %d, want 16", bps)
	}
}

func TestEncodeWAVEmpty(t *testing.T) {
	_, err := encodeWAV(nil)
	if err == nil {
		t.Error("expected error for empty samples")
	}
}

func TestEncodeWAVSize(t *testing.T) {
	samples := make([]int16, 16000) // 1 секунда
	wav, err := encodeWAV(samples)
	if err != nil {
		t.Fatal(err)
	}
	// 44 заголовок + 16000*2 = 32044
	expected := 44 + 16000*2
	if len(wav) != expected {
		t.Errorf("wav size = %d, want %d", len(wav), expected)
	}
}
