// Package correction — режим «Вася, поправь»: копирование выделенного текста,
// отправка в LLM-генератор правилки, вставка обратно в активное окно.
package correction

import (
	"context"
	"fmt"
	"strings"

	"github.com/vasya/protocol-voice-assistant/internal/llm"
	"github.com/vasya/protocol-voice-assistant/internal/logger"
	"github.com/vasya/protocol-voice-assistant/internal/paste"
)

// rulePrompt — системный промпт для правки (правила).
const rulePrompt = `Ты — помощник рентгенолога. Пользователь выделил фрагмент текста
и дал короткую инструкцию по его правке. Перепиши фрагмент согласно инструкции,
сохранив медицинскую точность и стиль. Верни ТОЛЬКО исправленный текст,
без пояснений, без кавычек, без markdown. Если инструкция противоречива —
сохрани исходный смысл. Не добавляй ничего от себя.`

// Service выполняет цикл правки выделенного текста.
type Service struct {
	llm   *llm.Client
	model string
	// копирование/вставка выделенного — через paste
}

// New создаёт сервис правки.
func New(llmClient *llm.Client, model string) *Service {
	return &Service{llm: llmClient, model: model}
}

// Correct выделяет текст (уже выделен пользователем), копирует его, правит
// через LLM и вставляет обратно.
// instruction — инструкция (например, "сделай короче").
func (s *Service) Correct(ctx context.Context, instruction string, researchID string) (*Result, error) {
	l := logger.Get()

	// 1. Копируем выделенное (если пункт выше выполнен пользователем, то
	//    выделение активно. Здесь мы просто читаем буфер).
	selected, err := paste.GetTextFromClipboard()
	if err != nil {
		return nil, fmt.Errorf("correction: read clipboard: %w", err)
	}
	if strings.TrimSpace(selected) == "" {
		return nil, fmt.Errorf("correction: буфер пуст — сначала выделите текст и скопируйте (Ctrl+C)")
	}
	if len(selected) > 8000 {
		s.LoopRejectedTooLong()
	}

	// 2. Отправляем в LLM.
	messages := []llm.Message{
		{Role: "system", Content: rulePrompt},
		{Role: "user", Content: fmt.Sprintf("Инструкция: %s\n\nВыделенный фрагмент:\n%s", instruction, selected)},
	}

	corrected, info, err := s.llm.Chat(llm.Request{
		Model:       s.model,
		Messages:    messages,
		Temperature: 0.2,
		MaxTokens:   2000,
	}, researchID)
	if err != nil {
		return nil, fmt.Errorf("correction: llm: %w", err)
	}

	corrected = strings.TrimSpace(corrected)

	// 3. Вставляем обратно в активное окно (заменяя выделенное) через Ctrl+V.
	res := &Result{
		Original:     selected,
		Corrected:    corrected,
		Instruction:  instruction,
		LlmLatencyMs: info.LatencyMs,
	}
	res.PasteRes = paste.SmartPaste(paste.ActiveWindow(), corrected, true)

	l.Event("correction_cycle", map[string]any{
		"research_id":      researchID,
		"selection_length": len(selected),
		"instruction":      instruction,
		"llm_latency_ms":   info.LatencyMs,
		"output_length":    len(corrected),
		"latency_total_ms": res.PasteRes.Latency,
	})
	l.Info(fmt.Sprintf("correction: правка завершена, length %d→%d", len(selected), len(corrected)))

	return res, nil
}

// LoopRejectedTooLong — заглушка для слишком длинного выделения.
func (s *Service) LoopRejectedTooLong() {}

// Result — результат цикла правки.
type Result struct {
	Original     string
	Corrected    string
	Instruction  string
	LlmLatencyMs int64
	PasteRes     paste.PasteResult
	// SelectionLen — длина выделенного.
	SelectionLen int
}
