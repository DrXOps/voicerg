package state

import (
	"sync"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// State — перечисление состояний конечного автомата.
type State int

const (
	Idle State = iota
	Recording
	Processing
	Done
	CommandRecording
	IntentProcessing
	NoteRecording
	NoteProcessing
)

var stateNames = map[State]string{
	Idle:             "Idle",
	Recording:        "Recording",
	Processing:       "Processing",
	Done:             "Done",
	CommandRecording: "CommandRecording",
	IntentProcessing: "IntentProcessing",
	NoteRecording:    "NoteRecording",
	NoteProcessing:   "NoteProcessing",
}

func (s State) String() string {
	if n, ok := stateNames[s]; ok {
		return n
	}
	return "Unknown"
}

// Event — правило перехода, регистрируемое в реестре.
type Event struct {
	From    []State
	To      State
	Trigger string // человекочитаемое описание для логов
}

// OnChange — callback на смену состояния (для трея/иконки).
type OnChange func(from, to State, trigger string)

// Machine — потокобезопасный конечный автомат с учётом времени пребывания
// в каждом состоянии (для хронометража).
type Machine struct {
	mu        sync.Mutex
	current   State
	entered   time.Time
	prevState State
	prevDur   time.Duration
	onChange  OnChange

	// История последних переходов (до 64) — для контекста intent-классификатора.
	history []HistoryEntry
}

type HistoryEntry struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Trigger string `json:"trigger"`
	At      string `json:"at"`
}

// New создаёт автомат, стартующий в Idle.
func New(onChange OnChange) *Machine {
	return &Machine{
		current:  Idle,
		entered:  time.Now(),
		onChange: onChange,
	}
}

// Current возвращает текущее состояние (потокобезопасно).
func (m *Machine) Current() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// Can переходит в состояние To, только если allowed содержит текущее.
// Возвращает (true, "") при успехе, или (false, reason).
func (m *Machine) Can(allowed []State, to State, trigger string) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.current
	for _, s := range allowed {
		if s == cur {
			// переход разрешён
			now := time.Now()
			dur := now.Sub(m.entered)
			from := m.current
			m.prevState = from
			m.prevDur = dur
			m.current = to
			m.entered = now
			m.appendHistory(from, to, trigger)
			lg := logger.Get()
			lg.Info("state " + from.String() + " -> " + to.String() + " (trigger=\"" + trigger + "\", dur=" + dur.Round(time.Millisecond).String() + ")")
			lg.Event("state_change", map[string]any{
				"from":                 from.String(),
				"to":                   to.String(),
				"trigger":              trigger,
				"trigger_source":       triggerSource(trigger),
				"duration_previous_ms": dur.Milliseconds(),
			})
			if m.onChange != nil {
				m.onChange(from, to, trigger)
			}
			return true, ""
		}
	}
	return false, "current=" + cur.String()
}

// DurationInCurrent возвращает, сколько уже находимся в текущем состоянии.
func (m *Machine) DurationInCurrent() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return time.Since(m.entered)
}

// History — последние N переходов (для intent-контекста).
func (m *Machine) History(n int) []HistoryEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n > len(m.history) {
		n = len(m.history)
	}
	out := make([]HistoryEntry, n)
	copy(out, m.history[len(m.history)-n:])
	return out
}

func (m *Machine) appendHistory(from, to State, trigger string) {
	entry := HistoryEntry{
		From:    from.String(),
		To:      to.String(),
		Trigger: trigger,
		At:      time.Now().Format(time.RFC3339),
	}
	m.history = append(m.history, entry)
	if len(m.history) > 64 {
		m.history = m.history[len(m.history)-64:]
	}
}

func triggerSource(t string) string {
	return "manual"
}
