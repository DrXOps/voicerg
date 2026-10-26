// Package hallucination — защита от галлюцинаций: после генерации протокола
// проверяем, что ключевые термины, присутствовавшие в диктовке врача, не
// пропали в ответе LLM. Если термин потерян — модель могла его «заменить».
package hallucination

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// Terms — реестр критических терминов (из critical_terms.json).
type Terms struct {
	List []string
}

// Load читает критичные термины из файла. Отсутствие файла не ошибка.
func Load(path string) (*Terms, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Terms{}, nil
		}
		return nil, fmt.Errorf("hallucination: read %s: %w", path, err)
	}

	var raw struct {
		Terms []string `json:"critical_terms"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("hallucination: parse %s: %w", path, err)
	}
	if raw.Terms == nil {
		// допускаем плоскую форму ["..."] тоже
		var flat []string
		if err := json.Unmarshal(data, &flat); err == nil {
			raw.Terms = flat
		}
	}
	for i := range raw.Terms {
		raw.Terms[i] = strings.ToLower(strings.TrimSpace(raw.Terms[i]))
	}
	return &Terms{List: raw.Terms}, nil
}

// Check проверяет, что все термины из dictation присутствуют в response
// (с учётом русских окончаний через префиксный матч по основе).
func (t *Terms) Check(dictation, response string) *Report {
	rep := &Report{CheckPassed: true}
	if t == nil || len(t.List) == 0 {
		return rep
	}

	dictLo := strings.ToLower(dictation)
	respLo := strings.ToLower(response)

	for _, term := range t.List {
		if term == "" {
			continue
		}
		// Термин встречается в диктовке?
		presentInInput := containsLexically(dictLo, term)
		if !presentInInput {
			continue // не было в диктовке — нечего проверять
		}
		// Есть ли он в ответе?
		presentInOutput := containsLexically(respLo, term)
		if !presentInOutput {
			rep.TermsMissing = append(rep.TermsMissing, term)
			rep.CheckPassed = false
		} else {
			rep.TermsFound = append(rep.TermsFound, term)
		}
	}
	return rep
}

// containsLexically — true, если term присутствует как подстрока, либо начало
// (префикс) любого слова в text (для учёта русских окончаний).
func containsLexically(text, term string) bool {
	if term == "" {
		return false
	}
	// 1. Точное вхождение подстроки.
	if strings.Contains(text, term) {
		return true
	}
	// 2. Префиксный матч: ищем слова, начинающиеся с термина.
	for _, word := range strings.FieldsFunc(text, isSepRune) {
		if len(word) >= len(term) && strings.HasPrefix(word, term) {
			return true
		}
	}
	return false
}

func isSepRune(r rune) bool {
	switch r {
	case ' ', ',', ';', ':', '.', '!', '?', '\t', '\n', '(', ')', '[', ']', '"', '\'', '—', '-':
		return true
	}
	return false
}

// Report — результат проверки.
type Report struct {
	TermsFound   []string `json:"terms_found_in_dictation"`
	TermsMissing []string `json:"terms_missing_in_response"`
	CheckPassed  bool     `json:"check_passed"`
}

// Log пишет событие hallucination_check в events.jsonl.
func (r *Report) Log() {
	l := logger.Get()
	l.Event("hallucination_check", map[string]any{
		"terms_found_in_dictation":  r.TermsFound,
		"terms_missing_in_response": r.TermsMissing,
		"check_passed":              r.CheckPassed,
		"warning_shown":             !r.CheckPassed,
	})
	if !r.CheckPassed {
		l.Warn(fmt.Sprintf("hallucination: потеряно терминов: %d (%s)",
			len(r.TermsMissing), strings.Join(r.TermsMissing, ", ")))
	}
}

// Terms иммутабелен после Load — мутекс не нужен.
