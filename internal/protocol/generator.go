// Package protocol — генерация рентгенологического протокола через Claude
// (RouterAI) на основе голосовой диктовки. Вики-промпт собирается из prompts/.
package protocol

import (
	"context"
	"fmt"
	"strings"

	"github.com/vasya/protocol-voice-assistant/internal/hotphrases"
	"github.com/vasya/protocol-voice-assistant/internal/llm"
	"github.com/vasya/protocol-voice-assistant/internal/prompt"
)

// hotExpander — интерфейс локальной подстановки (для тестируемости).
type hotExpander interface {
	Apply(text string) (string, int)
}

// Generator создаёт протокол из диктовки.
type Generator struct {
	llm      *llm.Client
	model    string
	wiki     *prompt.WikiBuilder
	mainFile string
	hot      hotExpander
}

// NewGenerator создаёт генератор протокола.
func NewGenerator(llmClient *llm.Client, model, mainFile string, wiki *prompt.WikiBuilder) *Generator {
	return &Generator{
		llm:      llmClient,
		model:    model,
		wiki:     wiki,
		mainFile: mainFile,
	}
}

// SetHotPhrases подключает реальное хранилище горячих фраз.
func (g *Generator) SetHotPhrases(h *hotphrases.Store) {
	g.hot = h
}

// Result — результат генерации протокола.
type Result struct {
	Text            string       `json:"text"`
	BlockParser     *BlockParser // для извлечения блоков
	TooMany         bool         `json:"-"`
	HotphrasesCount int          `json:"hotphrases_count"`
}

// Generate собирает вики-промпт, подставляет диктовку и вызывает Claude.
func (g *Generator) Generate(ctx context.Context, dictation string, researchID string) (*Result, error) {
	if g.wiki == nil {
		return nil, fmt.Errorf("protocol: wiki не задан")
	}

	systemPrompt, info, err := g.wiki.Build(g.mainFile)
	if err != nil {
		return nil, fmt.Errorf("protocol: build wiki: %w", err)
	}

	// Подстановка горячих фраз (локально, до отправки в LLM).
	processed, hotCount := expandHotPhrases(dictation)
	if g.hot != nil {
		var n int
		processed, n = g.hot.Apply(dictation)
		hotCount += n
	}
	if hotCount > 0 {
		_ = hotCount // лог уже внутри Apply
	}

	userMsg := "Диктовка исследования:\n\n" + processed + "\n\n" +
		"Сформируй структурированный протокол по правилам выше."

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}

	text, info2, err := g.llm.Chat(llm.Request{
		Model:       g.model,
		Messages:    messages,
		Temperature: 0.2,
	}, researchID)
	if err != nil {
		return nil, fmt.Errorf("protocol: claude: %w", err)
	}
	_ = info
	_ = info2

	sanitized := NormalizeProtocol(text)
	res := &Result{
		Text:            sanitized,
		HotphrasesCount: hotCount,
	}
	res.BlockParser = NewBlockParser(sanitized)
	return res, nil
}

// expandHotPhrases заменяет горячие фразы в тексте (локально).
// Реальная замена из hotphrases.json — на этапе 8. Пока заглушка.
func expandHotPhrases(text string) (string, int) {
	count := 0
	// базовые горячие фразы (встроенные запасные)
	replacements := map[string]string{
		"стандартное сердце": "Тень сердца обычной конфигурации, размеры не увеличены.",
		"чистые лёгкие":      "Легочный рисунок без патологических изменений.",
		"без патологии":      "Патологических изменений не выявлено.",
	}
	lower := strings.ToLower(text)
	for k, v := range replacements {
		// заменяем с учётом регистра? проще через lower-index
		idx := strings.Index(lower, k)
		if idx >= 0 {
			text = text[:idx] + v + text[idx+len(k):]
			lower = lower[:idx] + v + lower[idx+len(k):]
			count++
		}
	}
	return text, count
}

// NormalizeProtocol чистит ответ Claude: убирает markdown-ьюрные тройные кавычки.
func NormalizeProtocol(t string) string {
	t = strings.TrimSpace(t)
	// удаляем ``` ... ``` если вдруг
	if strings.HasPrefix(t, "```") {
		lines := strings.Split(t, "\n")
		if len(lines) > 2 {
			lines = lines[1 : len(lines)-1] // убираем первую и последнюю строки
			if strings.HasSuffix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			t = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(t)
}
