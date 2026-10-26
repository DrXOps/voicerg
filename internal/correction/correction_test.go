package correction

import (
	"strings"
	"testing"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

func initLog(t *testing.T) {
	t.Helper()
	_ = logger.Init(logger.Config{LogDir: t.TempDir(), LogFile: "t.log", MaxBytes: 1 << 20, KeepOld: 2})
}

// Тест построения промпта и логики отсечения пустого выделения.
// (LLM-вызов не тестируется без сети.)
func TestPromptText(t *testing.T) {
	initLog(t)
	if !strings.Contains(rulePrompt, "рентгенолога") {
		t.Error("rulePrompt не содержит контекст рентгенолога")
	}
}

func TestEmptySelection(t *testing.T) {
	initLog(t)
	// service без реального llm-клиента; проверка, что пустой буфер → err
	s := &Service{}
	res, err := s.Correct(nil, "сделай короче", "test-1")
	if err == nil {
		t.Error("ожидалась ошибка при пустом буфере")
	}
	if res != nil {
		t.Error("res должен быть nil при ошибке")
	}
}

func TestLoopRejectedTooLong(t *testing.T) {
	initLog(t)
	s := New(nil, "model")
	s.LoopRejectedTooLong() // не должно паниковать
}
