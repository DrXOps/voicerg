package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// Intent — структурированный результат классификации.
type Intent struct {
	Intent     string         `json:"intent"`
	Payload    map[string]any `json:"payload"`
	Confidence float64        `json:"confidence"`
}

// IntentClassifier — классификатор намерений через быструю LLM.
type IntentClassifier struct {
	client      *Client
	model       string
	prompt      string   // содержимое intent_prompt.md, кэшируется
	threshold   float64  // confidence threshold; ниже — не выполнять авто
	lastContext []string // последние 3 state transitions для контекста
}

// NewIntentClassifier создаёт классификатор.
func NewIntentClassifier(client *Client, model, promptFile string, threshold float64) *IntentClassifier {
	return &IntentClassifier{
		client:    client,
		model:     model,
		prompt:    promptFile,
		threshold: threshold,
	}
}

// Classify классифицирует текст: отправляет быстрый запрос (max_tokens=150)
// к LLM через RouterAI. Возвращает Intent.
// При ошибке или таймауте — fallback на локальную эвристику.
func (ic *IntentClassifier) Classify(text string, contextActions []string, requestID string) (*Intent, error) {
	l := logger.Get()
	start := time.Now()

	messages := []Message{
		{Role: "system", Content: ic.prompt},
		{Role: "user", Content: ic.buildQuery(text, contextActions)},
	}

	resp, info, err := ic.client.Chat(Request{
		Model:       ic.model,
		Messages:    messages,
		MaxTokens:   150,
		Temperature: 0.0,
	}, requestID)

	latency := time.Since(start)

	// Fallback: если модель не ответила или таймаут — локальная эвристика
	if err != nil || info == nil {
		l.Warn(fmt.Sprintf("intent: LLM не ответил (%v), fallback на эвристику", err))
		return ic.fallbackHeuristic(text), nil
	}

	// Парсим JSON-ответ (ожидаем {"intent", "payload", "confidence"})
	intent, parseErr := ic.parseResponse(resp)
	if parseErr != nil {
		l.Warn("intent: не удалось распарсить ответ LLM: " + parseErr.Error())
		return ic.fallbackHeuristic(text), nil
	}

	l.Info(fmt.Sprintf("intent: %s, confidence=%.2f, latency=%v",
		intent.Intent, intent.Confidence, latency.Round(time.Millisecond)))
	l.Event("intent_response", map[string]any{
		"intent":     intent.Intent,
		"payload":    intent.Payload,
		"confidence": intent.Confidence,
		"latency_ms": latency.Milliseconds(),
		"request_id": requestID,
	})

	return intent, nil
}

// SetSystemPrompt обновляет системный промпт (при перезагрузке intent_prompt.md).
func (ic *IntentClassifier) SetSystemPrompt(prompt string) {
	ic.prompt = prompt
}

// SetContext обновляет историю действий.
func (ic *IntentClassifier) SetContext(actions []string) {
	ic.lastContext = actions
}

func (ic *IntentClassifier) buildQuery(text string, contextActions []string) string {
	q := fmt.Sprintf("Текст: %s\nКонтекст: ", text)
	for i, a := range contextActions {
		if i >= 3 {
			break
		}
		q += a + "; "
	}
	if len(contextActions) == 0 {
		q += "(пусто)"
	}
	return q
}

func (ic *IntentClassifier) parseResponse(resp string) (*Intent, error) {
	// LLM может завернуть в ```json ... ``` — извлекаем
	cleaned := stripMarkdown(resp)
	var intent Intent
	if err := json.Unmarshal([]byte(cleaned), &intent); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	if intent.Intent == "" {
		return nil, fmt.Errorf("intent empty")
	}
	return &intent, nil
}

// fallbackHeuristic — локальная эвристика по маркерным словам (без LLM).
func (ic *IntentClassifier) fallbackHeuristic(text string) *Intent {
	// Простейший fallback: по ключевым словам
	keywords := map[string]struct {
		intent  string
		payload map[string]any
	}{
		"поехали":    {"start_recording", map[string]any{"profile": "knee"}},
		"пишем":      {"start_recording", map[string]any{"profile": "default"}},
		"начинаем":   {"start_recording", map[string]any{"profile": "default"}},
		"заключение": {"stop_recording", map[string]any{}},
		"закончил":   {"stop_recording", map[string]any{}},
		"отмена":     {"cancel", map[string]any{}},
		"забудь":     {"cancel", map[string]any{}},
	}
	lowText := strings.ToLower(text)
	for kw, res := range keywords {
		if strings.Contains(lowText, kw) {
			return &Intent{Intent: res.intent, Payload: res.payload, Confidence: 0.7}
		}
	}
	return &Intent{Intent: "unknown", Payload: map[string]any{}, Confidence: 0.0}
}

// stripMarkdown вырезает ```json ... ``` маркеры.
func stripMarkdown(s string) string {
	if len(s) < 8 {
		return s
	}
	prefix := "```json\n"
	if strings.HasPrefix(s, prefix) {
		s = s[len(prefix):]
	}
	suffix := "\n```"
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}
