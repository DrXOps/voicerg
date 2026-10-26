package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
	"github.com/vasya/protocol-voice-assistant/internal/paste"
	"github.com/vasya/protocol-voice-assistant/internal/state"
	"github.com/vasya/protocol-voice-assistant/internal/tray"
)

// IdleHandler — быстрый путь (wake)
type IdleHandler struct {
	*handlerBase
	router *Router
	openFn func(string)
}

func NewIdleHandler(r *Router, openFn func(string)) *IdleHandler {
	return &IdleHandler{
		handlerBase: &handlerBase{name: "idle", types: []string{"wake", "intent"}},
		router:      r,
		openFn:      openFn,
	}
}

func (h *IdleHandler) Match(ev Event) bool {
	return h.matches(ev) && h.router.sm.Current() == state.Idle
}

func (h *IdleHandler) Handle(ctx context.Context, ev Event) error {
	l := logger.Get()
	if ev.Type == "wake" {
		// Быстрый путь: переходим в Recording
		if ok, _ := h.router.sm.Can([]state.State{state.Idle}, state.Recording, fmt.Sprintf("wake:%s", ev.Phrase)); !ok {
			return fmt.Errorf("idle: cannot transition from %s", h.router.sm.Current())
		}
		l.Info("IdleHandler: wake -> Recording, profile=" + ev.Profile)
		// TODO: сигнал для DictationHandler, что собрать профиль и начать запись
		return nil
	}
	if ev.Type == "intent" && ev.Intent == "start_recording" {
		prof, _ := ev.Payload["profile"].(string)
		if prof == "" {
			prof = "default"
		}
		if ok, _ := h.router.sm.Can([]state.State{state.Idle}, state.Recording, fmt.Sprintf("intent:%s", ev.Intent)); !ok {
			return nil
		}
		l.Info(fmt.Sprintf("IdleHandler: intent recording, profile=%s", prof))
		return nil
	}
	return nil
}

// DictationHandler — диктовка → протокол → вставка.
type DictationHandler struct {
	*handlerBase
	router    *Router
	generator interface {
		Generate(ctx context.Context, dictation, researchID string) (string, int, string, error)
	}
}

func NewDictationHandler(r *Router) *DictationHandler {
	return &DictationHandler{
		handlerBase: &handlerBase{name: "dictation", types: []string{"stop"}},
		router:      r,
	}
}

func (h *DictationHandler) Match(ev Event) bool {
	return h.matches(ev) && h.router.sm.Current() == state.Recording
}

func (h *DictationHandler) Handle(ctx context.Context, ev Event) error {
	l := logger.Get()
	if ok, _ := h.router.sm.Can([]state.State{state.Recording}, state.Processing, "stop"); !ok {
		return fmt.Errorf("dictation: not in recording")
	}

	// TODO: здесь должен быть вызов Whisper (STT) и немного сложнее логика.
	l.Info("DictationHandler: Recording -> Processing")
	// Симуляция: ожидаем, что результат пришёл
	_ = ev

	// переходим в Done
	h.router.sm.Can([]state.State{state.Processing}, state.Done, "done")

	// Умная вставка
	w := paste.ActiveWindow()
	res := paste.SmartPaste(w, "(протокол будет здесь)", true)
	if res.Pasted {
		l.Info("dictation: протокол вставлен в «" + w.Title + "»")
	} else {
		l.Info("dictation: протокол в буфере (вставьте Ctrl+V)")
	}

	// Возврат в Idle через секунду
	go func() {
		time.Sleep(2 * time.Second)
		h.router.sm.Can([]state.State{state.Done}, state.Idle, "auto-reset")
	}()

	return nil
}

// NoteHandler — голосовые заметки.
type NoteHandler struct {
	*handlerBase
	router *Router
}

func NewNoteHandler(r *Router) *NoteHandler {
	return &NoteHandler{
		handlerBase: &handlerBase{name: "note", types: []string{"note", "intent"}},
		router:      r,
	}
}

func (h *NoteHandler) Match(ev Event) bool {
	if !h.matches(ev) {
		return false
	}
	cur := h.router.sm.Current()
	switch cur {
	case state.Idle:
		return true
	case state.Recording:
		return true // можно из Recording переключиться на NoteRecording
	default:
		return false
	}
}

func (h *NoteHandler) Handle(ctx context.Context, ev Event) error {
	l := logger.Get()
	cur := h.router.sm.Current()

	if cur == state.Idle || cur == state.Recording {
		h.router.sm.Can([]state.State{cur}, state.NoteRecording, "note-wake")
		l.Info("NoteHandler: -> NoteRecording")
	}

	// Симуляция остановки записи заметки
	go func() {
		time.Sleep(3 * time.Second)
		h.router.sm.Can([]state.State{state.NoteRecording}, state.NoteProcessing, "note-stop")
		l.Info("NoteHandler: NoteRecording -> NoteProcessing")

		// TODO: STT + LLM + запись

		// восстановление
		h.router.sm.Can([]state.State{state.NoteProcessing}, state.Idle, "note-done")
		l.Info("NoteHandler: заметка сохранена")
	}()

	return nil
}

// CorrectionHandler — «Вася, поправь».
type CorrectionHandler struct {
	*handlerBase
	router  *Router
	service interface {
		Correct(ctx context.Context, instruction, researchID string) error
	}
}

func NewCorrectionHandler(r *Router) *CorrectionHandler {
	return &CorrectionHandler{
		handlerBase: &handlerBase{name: "correction", types: []string{"intent"}},
		router:      r,
	}
}

func (h *CorrectionHandler) Match(ev Event) bool {
	if !h.matches(ev) {
		return false
	}
	return ev.Intent == "correct"
}

func (h *CorrectionHandler) Handle(ctx context.Context, ev Event) error {
	l := logger.Get()
	instruction, _ := ev.Payload["instruction"].(string)
	if instruction == "" {
		instruction = "поправь текст"
	}
	l.Info("CorrectionHandler: правка «" + instruction + "»")
	// TODO: запрос к correction.Service
	_ = instruction
	return nil
}

// Импорт tray, paste, fmt требуется для других частей кода (ссылки на типы).
var _ = fmt.Println
var _ = tray.OpenPath
var _ = paste.SmartPaste
