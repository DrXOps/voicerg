package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/audio"
	"github.com/vasya/protocol-voice-assistant/internal/config"
	"github.com/vasya/protocol-voice-assistant/internal/icons"
	"github.com/vasya/protocol-voice-assistant/internal/logger"
	"github.com/vasya/protocol-voice-assistant/internal/state"
	"github.com/vasya/protocol-voice-assistant/internal/tray"
)

// iconsGen генерирует fallback-набор иконок в каталог icons/ рядом с ресурсами.
func iconsGen(dir string) ([]string, error) {
	return icons.Save(dir)
}

var (
	version   = "0.1.0"
	buildTime = "dev"
)

func main() {
	// Каталог рядом с exe — точка отсчёта всех путей.
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	workDir := filepath.Dir(exePath)

	// --- Инициализация логгера (два файла + сессии).
	lg := logger.Init(logger.Config{
		LogDir:   filepath.Join(workDir, "logs"),
		LogFile:  "protocol-assistant.log",
		MaxBytes: 10 << 20,
		KeepOld:  5,
	})

	host, _ := os.Hostname()
	appStart := time.Now()

	lg.Info(fmt.Sprintf("=== Вася %s (%s) стартует на %s/%s host=%s ===",
		version, buildTime, runtime.GOOS, runtime.GOARCH, host))
	lg.Event("app_start", map[string]any{
		"version":    version,
		"os":         runtime.GOOS,
		"hostname":   host,
		"started":    appStart.Format(time.RFC3339),
		"session_id": lg.SessionID(),
	})

	// --- Конфигурация.
	cfg, err := config.Load(workDir)
	if err != nil {
		lg.Error("config load failed: " + err.Error())
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	lg.Event("config_loaded", map[string]any{
		"path": filepath.Join(workDir, "config.json"),
		"hash": cfg.Hash(),
		"params": map[string]any{
			"device_name":       cfg.DeviceName,
			"prompts_dir":       cfg.PromptsDir,
			"smart_paste":       cfg.SmartPaste,
			"intent_model":      cfg.IntentModel,
			"stt_model":         cfg.STTModel,
			"claude_model":      cfg.ClaudeModel,
			"routerai_base_url": cfg.RouterAIBaseURL,
			"routerai_key":      "***", // никогда не логируем секрет
			"vosk_model_path":   cfg.VoskModelPath,
			"wake_words_count":  len(cfg.WakeWords),
			"stop_words_count":  len(cfg.StopWords),
		},
	})
	lg.Info("config loaded, hash=" + cfg.Hash())

	// --- Контекст всего приложения (для graceful shutdown).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Иконки: генерируем программно при первом запуске (fallback-набор).
	iconFiles, iconErr := iconsGen(cfg.WorkingDir())
	if iconErr != nil {
		lg.Warn("icons generation: " + iconErr.Error())
	} else {
		lg.Info(fmt.Sprintf("icons ready: %d файлов", len(iconFiles)))
	}

	// --- Конечный автомат.
	sm := state.New(func(from, to state.State, trigger string) {
		lg.Info("state " + from.String() + " -> " + to.String() + " trigger=" + trigger)
	})

	// --- Аудио: перечисление устройств (для app_start) + захват.
	audioSrc := audio.NewSource()
	devices, devErr := audioSrc.ListDevices()
	var devNames []string
	for _, d := range devices {
		devNames = append(devNames, d.Name)
	}
	if devErr != nil {
		lg.Warn("audio: не удалось перечислить устройства: " + devErr.Error())
	}
	lg.Event("audio_devices", map[string]any{
		"devices":         devNames,
		"selected_device": cfg.DeviceName,
		"count":           len(devNames),
	})
	lg.Info(fmt.Sprintf("audio: найдено устройств=%d, выбрано=%q", len(devNames), cfg.DeviceName))

	// старт захвата (на Windows — реальный, на Linux — заглушка вернёт ошибку)
	audioCtx, audioCancel := context.WithCancel(ctx)
	defer audioCancel()
	if err := audioSrc.Start(audioCtx, cfg.DeviceName); err != nil {
		lg.Warn("audio: захват не запущен (этап без аудио-устройства?): " + err.Error())
	} else {
		lg.Info("audio: захват активен, канал готов")
		go monitorAudio(audioSrc, lg)
	}

	// --- Трей.
	app := tray.New(cfg, sm, openPath)

	// Запуск трея в отдельной горутине (systray.Run блокирует).
	go func() {
		app.Run(ctx)
		cancel()
	}()

	lg.Info("tray started")

	// --- Ожидание выхода.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		lg.Info("signal received, shutting down")
	case <-ctx.Done():
	}

	uptime := time.Since(appStart)
	lg.Event("app_stop", map[string]any{
		"uptime_seconds": int(uptime.Seconds()),
		"session_id":     lg.SessionID(),
	})
	// фиксируем summary сессии (заглушка, наполнится последующими этапами)
	writeSessionSummary(lg, workDir)
	lg.Info(fmt.Sprintf("=== Вася остановлен, uptime=%s ===", uptime.Round(time.Second)))
}

// openPath открывает папку/файл в системном файловом менеджере.
func openPath(p string) {
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
