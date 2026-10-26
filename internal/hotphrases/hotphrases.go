// Package hotphrases — локальная подстановка горячих фраз перед отправкой в LLM.
// Загружает hotphrases.json (рядом с exe), заменяет известные сокращения в
// диктовке на полные формулировки, экономя обращение к LLM для типовых блоков.
package hotphrases

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// Store — реестр горячих фраз.
type Store struct {
	mu    sync.RWMutex
	items map[string]string
	path  string
}

// New создаёт пустое хранилище; path — путь к hotphrases.json.
func New(path string) *Store {
	return &Store{items: map[string]string{}, path: path}
}

// Load читает hotphrases.json из path. Если файл отсутствует — не ошибка
// (просто пустое хранилище). Ключи нормализуются в нижний регистр.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			l := logger.Get()
			l.Warn("hotphrases: файл не найден, используется пустое хранилище: " + s.path)
			return nil
		}
		return fmt.Errorf("hotphrases: read %s: %w", s.path, err)
	}

	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("hotphrases: parse %s: %w", s.path, err)
	}

	normalized := make(map[string]string, len(raw))
	for k, v := range raw {
		normalized[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	s.items = normalized

	l := logger.Get()
	l.Info(fmt.Sprintf("hotphrases: загружено %d фраз из %s", len(normalized), s.path))
	return nil
}

// Apply заменяет горячие фразы в тексте. Возвращает обработанный текст и
// количество заменённых фраз. Логирует метрику.
func (s *Store) Apply(text string) (string, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.items) == 0 || text == "" {
		return text, 0
	}

	lower := strings.ToLower(text)
	count := 0
	// Заменяем по убыванию длины (чтобы длинные фразы приоритетнее коротких,
	// например "без патологии" → полный текст).
	keys := sortedByLen(s.items)

	for _, k := range keys {
		v := s.items[k]
		if v == "" {
			continue
		}
		// заменяем все вхождения (регистронезависимо)
		var sb strings.Builder
		rest := text
		restLower := lower
		for {
			idx := strings.Index(restLower, k)
			if idx < 0 {
				sb.WriteString(rest)
				break
			}
			sb.WriteString(rest[:idx])
			sb.WriteString(v)
			rest = rest[idx+len(k):]
			restLower = restLower[idx+len(k):]
			count++
		}
		text = sb.String()
		lower = strings.ToLower(text)
	}

	if count > 0 {
		l := logger.Get()
		l.Event("hotphrase_substituted", map[string]any{
			"phrase":              "composite",
			"replacement_count":   count,
			"replacement_length":  len(text),
			"savings_estimate_ms": count * 1500, // ~1.5s экономии на фразу
		})
		// вычислим, сколько конкретных фраз заменено (для точного лога)
		l.Info(fmt.Sprintf("hotphrases: заменено фраз: %d", count))
	}

	return text, count
}

// Keys возвращает ключи, отсортированные по убыванию длины.
func sortedByLen(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// сортировка вставками по длине (убывание)
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && len(keys[j-1]) < len(keys[j]); j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
