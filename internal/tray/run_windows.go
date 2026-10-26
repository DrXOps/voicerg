//go:build windows

package tray

import (
	"context"
	"fmt"

	"github.com/getlantern/systray"

	"github.com/vasya/protocol-voice-assistant/internal/icons"
	"github.com/vasya/protocol-voice-assistant/internal/state"
)

// Run запускает реальный систray (Windows). Блокирует до Quit.
func (a *App) Run(ctx context.Context) {
	a.l.Info("tray: starting systray")
	systray.Run(a.onReady, a.onExit)
	a.l.Info("tray: stopped")
	<-ctx.Done()
}

func (a *App) onReady() {
	idle, err := icons.LoadOrFallback(a.cfg.WorkingDir(), icons.KindIdle)
	if err == nil {
		systray.SetIcon(idle)
	}
	systray.SetTitle("Вася — протокол")
	systray.SetTooltip("Вася — голосовой ассистент рентгенолога")

	mStatus := systray.AddMenuItem("Показать статус", "Текущее состояние")
	mPrompts := systray.AddMenuItem("Открыть папку промптов", "prompts/")
	mNotes := systray.AddMenuItem("Открыть папку заметок", "notes/")
	mLogsSub := systray.AddMenuItem("Открыть логи и метрики", "logs/")
	mEvents := mLogsSub.AddSubMenuItem("Открыть events.jsonl", "events.jsonl")
	mSessions := mLogsSub.AddSubMenuItem("Открыть последний session_summary", "sessions/")
	mLogFile := mLogsSub.AddSubMenuItem("Открыть protocol-assistant.log", "protocol-assistant.log")
	mConfig := systray.AddMenuItem("Открыть config.json", "config.json")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Завершить работу")

	go func() {
		for {
			select {
			case <-mStatus.ClickedCh:
				cur := a.sm.Current()
				a.Notify("Вася", "Статус: "+cur.String())
			case <-mPrompts.ClickedCh:
				OpenPath(a.Resolve(a.cfg.PromptsDir))
			case <-mNotes.ClickedCh:
				OpenPath(a.Resolve("notes"))
			case <-mLogsSub.ClickedCh:
				OpenPath(a.Resolve("logs"))
			case <-mEvents.ClickedCh:
				OpenPath(a.Resolve("logs/events.jsonl"))
			case <-mSessions.ClickedCh:
				OpenPath(a.Resolve("logs/sessions"))
			case <-mLogFile.ClickedCh:
				OpenPath(a.Resolve("logs/protocol-assistant.log"))
			case <-mConfig.ClickedCh:
				OpenPath(a.Resolve("config.json"))
			case <-mQuit.ClickedCh:
				a.Quit()
				return
			}
		}
	}()
}

func (a *App) onExit() {
	a.l.Info("tray: onExit")
}

func (a *App) notify(title, msg string) {
	// systray v1.2.2 не имеет public API для пузырьковых уведомлений; пока
	// показываем индикатор через tooltip + логируем. Полноценный balloon-tip
	// через Shell_NotifyIcon добавим на этапе умной вставки.
	systray.SetTooltip(title + " — " + msg)
	a.l.Info(fmt.Sprintf("notify [%s] %s", title, msg))
}

func (a *App) quit() {
	a.l.Info("tray: quit requested")
	systray.Quit()
}

func (a *App) setStateIcon(s state.State) {
	ico, err := icons.LoadOrFallback(a.cfg.WorkingDir(), icons.KindForState(s))
	if err != nil {
		return
	}
	systray.SetIcon(ico)
	_ = fmt.Sprintf("%v", s) // неиспользуемый import-фикс
}
