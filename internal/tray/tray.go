// Package tray — системный трей и меню. Платформонезависимая часть: структура
// App, методы открытия файла. Фактический трей-рантайм разделён по платформам:
//   - windows: реальный systray (internal/tray/run_windows.go)
//   - linux/macos: заглушка (internal/tray/run_stub.go), т.к. systray на linux
//     требует ayatana-appindicator (GTK), недоступный в headless-окружении.
package tray

import (
	"os/exec"
	"runtime"

	"github.com/vasya/protocol-voice-assistant/internal/config"
	"github.com/vasya/protocol-voice-assistant/internal/logger"
	"github.com/vasya/protocol-voice-assistant/internal/state"
)

// App — инкапсулирует трей и меню.
type App struct {
	cfg    *config.Config
	sm     *state.Machine
	onOpen func(dir string)
	l      *logger.Logger
}

// New создаёт объект трея.
func New(cfg *config.Config, sm *state.Machine, openDir func(string)) *App {
	return &App{
		cfg:    cfg,
		sm:     sm,
		onOpen: openDir,
		l:      logger.Get(),
	}
}

// Notify показывает всплывающее уведомление (платформозависимо).
// Export через трамп по build tags.
func (a *App) Notify(title, msg string) { a.notify(title, msg) }

// Quit завершает работу трея.
func (a *App) Quit() { a.quit() }

// SetStateIcon меняет иконку в трее по состоянию.
func (a *App) SetStateIcon(s state.State) { a.setStateIcon(s) }

// Resolve — абсолютный путь к ресурсу из рабочей директории.
func (a *App) Resolve(p string) string { return a.cfg.Resolve(p) }

func (a *App) workDir() string { return a.cfg.WorkingDir() }

// OpenPath открывает папку/файл в системном менеджере (общий для платформ).
func OpenPath(p string) {
	var cmd string
	switch runtime.GOOS {
	case "windows":
		cmd = "explorer"
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	_ = exec.Command(cmd, p).Start()
}
