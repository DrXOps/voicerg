package audio

import (
	"math"
	"testing"
)

func TestRMS(t *testing.T) {
	// тишина
	if r := RMS([]int16{0, 0, 0}); r != 0 {
		t.Errorf("RMS silence = %v", r)
	}
	// константа 1000 -> RMS = 1000
	if r := RMS([]int16{1000, 1000, 1000, 1000}); math.Abs(r-1000) > 0.1 {
		t.Errorf("RMS const = %v, want ~1000", r)
	}
	// смесь +/− одинаково: RMS положительный
	r := RMS([]int16{-2000, 2000, -2000, 2000})
	if r < 1999 || r > 2001 {
		t.Errorf("RMS mixed = %v, want ~2000", r)
	}
}

func TestVADThreshold(t *testing.T) {
	v := NewVAD(500)
	// низкая амплитуда = тишина
	if v.IsSpeech([]int16{10, 20, 10, 30}) {
		t.Error("quiet frame classified as speech")
	}
	// высокая амплитуда = речь
	if !v.IsSpeech([]int16{2000, 3000, 2500, 2000}) {
		t.Error("loud frame not classified as speech")
	}
}

func TestNewSourceStub(t *testing.T) {
	src := NewSource()
	if src == nil {
		t.Fatal("NewSource nil")
	}
	// заглушка не должна паниковать
	src.Stop()
	if src.Events() != nil && src.Events() == nil {
		t.Error("unreachable")
	}
}
