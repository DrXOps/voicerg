// Package engine — центральный CommandRouter: связывает модули (аудио, вики,
// похождение интентов, вставку) в целостный рабочий цикл.
package engine

import (
	"context"
	"sync"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
	"github.com/vasya/protocol-voice-assistant/internal/state"
)

// Event — событие из Vosk/интентов, которое передаётся хендлерам.
type Event struct {
	// Type — тип события: "wake" | "stop" | "note" | "intent"
	Type string
	// Phrase — распознанная фраза.
	Phrase string
	// Profile — профиль исследования (knee/chest/...).
	Profile string
	// Intent — имя интента (для intent-событий).
	Intent string
	// Payload — данные интента.
	Payload map[string]any
}

// Handler — интерфейс хендлера команды.
type Handler interface {
	// Name — имя хендлера (для логов).
	Name() string
	// Match — может ли этот хендлер обработать событие.
	Match(ev Event) bool
	// Handle — обработка события.
	Handle(ctx context.Context, ev Event) error
}

// Router — центральный диспетчер.
type Router struct {
	handlers []Handler
	sm       *state.Machine
	mu       sync.Mutex
}

// NewRouter создаёт диспетчер с конечным автоматом.
func NewRouter(sm *state.Machine) *Router {
	return &Router{sm: sm}
}

// Register добавляет хендлер в реестр.
func (r *Router) Register(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, h)
}

// Dispatch роутит событие пакет-хендлеру.
func (r *Router) Dispatch(ctx context.Context, ev Event) {
	l := logger.Get()
	r.mu.Lock()
	handlers := make([]Handler, len(r.handlers))
	copy(handlers, r.handlers)
	r.mu.Unlock()

	for _, h := range handlers {
		if h.Match(ev) {
			if err := h.Handle(ctx, ev); err != nil {
				l.Error("router[" + h.Name() + "]: " + err.Error())
			}
			return
		}
	}
	l.Warn("router: нет подходящего хендлера для события " + ev.Type)
}

// StateMachine возвращает автомат (для внешних сервисов).
func (r *Router) StateMachine() *state.Machine { return r.sm }

// Handlers возвращает зарегистрированные хендлеры.
func (r *Router) Handlers() []Handler {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := make([]Handler, len(r.handlers))
	copy(h, r.handlers)
	return h
}

// handlerBase — общий прайм для хендлеров.
type handlerBase struct {
	name  string
	types []string
}

func (h *handlerBase) Name() string { return h.name }

func (h *handlerBase) matches(ev Event) bool {
	for _, t := range h.types {
		if t == ev.Type || t == "*" {
			return true
		}
	}
	return false
}
