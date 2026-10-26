package protocol

import (
	"strings"
	"testing"
)

func TestBlockParser(t *testing.T) {
	text := `[Название]
Протокол рентгенографии коленного сустава

[ОПИСАНИЕ]
Суставная щель сохранена.

[ЗАКЛЮЧЕНИЕ]
Дегенеративно-дистрофических изменений не выявлено.

[РЕКОМЕНДАЦИИ]
Динамическое наблюдение.`

	bp := NewBlockParser(text)

	// ЗАКЛЮЧЕНИЕ
	z := bp.GetBlock("ЗАКЛЮЧЕНИЕ")
	if !strings.Contains(z, "Дегенеративно") {
		t.Errorf("заключение блок = %q", z)
	}
	if strings.Contains(z, "РЕКОМЕНДАЦИИ") {
		t.Error("заключение не должно содержать рекомендации")
	}

	// ОПИСАНИЕ
	o := bp.GetBlock("ОПИСАНИЕ")
	if !strings.Contains(o, "Суставная щель") {
		t.Errorf("описание блок = %q", o)
	}

	// несуществующий
	if bp.GetBlock("КЛИНИЧЕСКИЕ ДАННЫЕ") != "" {
		t.Error("несуществующий блок должен быть пустым")
	}
}

func TestNormalizeProtocol(t *testing.T) {
	in := "```json\n[Название]\nТест\n```"
	out := NormalizeProtocol(in)
	if strings.Contains(out, "```") {
		t.Errorf("markdown-квотинг не убран: %q", out)
	}
	if !strings.Contains(out, "[Название]") {
		t.Errorf("маркер пропал: %q", out)
	}
}

func TestExpandHotPhrases(t *testing.T) {
	out, n := expandHotPhrases("Чистые лёгкие, без патологии")
	if n != 2 {
		t.Errorf("заменено %d, want 2", n)
	}
	if !strings.Contains(out, "Легочный рисунок") {
		t.Errorf("чистые лёгкие не заменены: %q", out)
	}
}