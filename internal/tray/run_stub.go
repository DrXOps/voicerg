//go:build !windows

package tray

import (
	"context"
	"fmt"

	"github.com/vasya/protocol-voice-assistant/internal/state"
)

// Run — заглушка: systray не доступен без GTK, поэтому на linux/macos просто
// логируем и держим достойный цикл (для тестирования логики).
func (a *App) Run(ctx context.Context) {
	a.l.Info("tray: RUN STUB (not windows) — tray disabled, state machine live")
	cur := a.sm.Current()
	a.l.Info("tray stub: current state = " + cur.String())
	<-ctx.Done()
	a.l.Info("tray: stub ctx done")
}

func (a *App) notify(title, msg string) {
	a.l.Info(fmt.Sprintf("notify (stub) [%s] %s", title, msg))
}

func (a *App) quit() {
	a.l.Info("tray: quit requested (stub)")
}

func (a *App) setStateIcon(s state.State) {
	a.l.Info("tray: setStateIcon stub -> " + s.String())
}
