package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config настройки ротации.
type Config struct {
	LogDir   string
	LogFile  string // protocol-assistant.log
	MaxBytes int64  // ротация по размеру
	KeepOld  int    // сколько старых файлов хранить
}

// Logger — thread-safe single instance для двух логов:
//   - человекочитаемый protocol-assistant.log с ротацией;
//   - machine-readable events.jsonl (JSON Lines) с ротацией.
type Logger struct {
	mu        sync.Mutex
	cfg       Config
	human     *os.File
	events    *os.File
	humanSize int64
	evSize    int64
	sessionID string
	started   time.Time
}

var (
	instance *Logger
	muInit   sync.Mutex
)

// resetForTest пересоздаёт инстанс (только для тестов).
func resetForTest() {
	muInit.Lock()
	defer muInit.Unlock()
	if instance != nil && instance.human != nil {
		_ = instance.human.Close()
		_ = instance.events.Close()
	}
	instance = nil
}

// Init создаёт (или возвращает существующий) single-instance логгер.
func Init(cfg Config) *Logger {
	muInit.Lock()
	defer muInit.Unlock()
	if instance != nil {
		return instance
	}
	_ = os.MkdirAll(cfg.LogDir, 0o755)
	h, hs := openRotated(filepath.Join(cfg.LogDir, cfg.LogFile))
	e, es := openRotated(filepath.Join(cfg.LogDir, "events.jsonl"))
	instance = &Logger{
		cfg:       cfg,
		human:     h,
		events:    e,
		humanSize: hs,
		evSize:    es,
		sessionID: newUUID(),
		started:   time.Now(),
	}
	return instance
}

// Get возвращает единственный инстанс (паник-фолбэк на дефолт, если не Init).
func Get() *Logger {
	if instance == nil {
		return Init(Config{LogDir: ".", LogFile: "protocol-assistant.log", MaxBytes: 10 << 20, KeepOld: 5})
	}
	return instance
}

// SessionID — UUID текущей сессии.
func (l *Logger) SessionID() string {
	mu := &l.mu
	mu.Lock()
	defer mu.Unlock()
	return l.sessionID
}

// StartedAt — время старта приложения.
func (l *Logger) StartedAt() time.Time { return l.started }

// Info — человекочитаемое сообщение уровня INFO.
func (l *Logger) Info(msg string) {
	l.writeHuman("INFO", msg)
}

// Debug — уровень DEBUG.
func (l *Logger) Debug(msg string) {
	l.writeHuman("DEBUG", msg)
}

// Warn — уровень WARN.
func (l *Logger) Warn(msg string) {
	l.writeHuman("WARN", msg)
}

// Error — уровень ERROR.
func (l *Logger) Error(msg string) {
	l.writeHuman("ERROR", msg)
}

func (l *Logger) writeHuman(level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("%s [%s] %s\n", ts, level, msg)
	n, _ := l.human.WriteString(line)
	l.humanSize += int64(n)
	l.maybeRotateHuman()
}

// Event пишет типизированное событие в events.jsonl.
// data сериализуется как JSON без потерь.
func (l *Logger) Event(event string, data map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec := map[string]any{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"event": event,
		"data":  data,
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	b = append(b, '\n')
	n, _ := l.events.Write(b)
	l.evSize += int64(n)
	l.maybeRotateEvents()
}

// Timed возвращает defer-функцию для замера длительности операции >10 мс.
// Пример: defer logger.Timed("stt.request", researchID)().
func Timed(component, id string) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		Get().Info(fmt.Sprintf("%s [%s] done in %s", component, id, elapsed))
	}
}

func (l *Logger) maybeRotateHuman() {
	if l.humanSize < l.cfg.MaxBytes {
		return
	}
	p := filepath.Join(l.cfg.LogDir, l.cfg.LogFile)
	_ = l.human.Close()
	rotateBackup(p, l.cfg.KeepOld)
	h, n := openRotated(p)
	l.human = h
	l.humanSize = n
}

func (l *Logger) maybeRotateEvents() {
	const evMax = 50 << 20
	if l.evSize < evMax {
		return
	}
	p := filepath.Join(l.cfg.LogDir, "events.jsonl")
	_ = l.events.Close()
	rotateBackup(p, 10)
	e, n := openRotated(p)
	l.events = e
	l.evSize = n
}

func openRotated(path string) (*os.File, int64) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		// fallback: без файла, пишем в никуда
		return os.NewFile(0, path), 0
	}
	st, _ := f.Stat()
	return f, st.Size()
}

func rotateBackup(path string, keep int) {
	// сдвигаем .1 -> .2 ... .N
	for i := keep; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", path, i)
		to := fmt.Sprintf("%s.%d", path, i+1)
		_ = os.Rename(from, to)
	}
	_ = os.Rename(path, path+".1")
}

func newUUID() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		rand32(), rand16(), rand16(), rand16(), rand32())
}

func rand16() uint32 { return rand32() & 0xffff }
func rand32() uint32 {
	return uint32(time.Now().UnixNano()&0xffffffff) ^ uint32(os.Getpid()<<16>>16)
}
