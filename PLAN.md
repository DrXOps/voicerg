# План разработки «Вася» — голосовой ассистент рентгенолога

> Фоновое Windows-приложение в системном трее. Работаю поэтапно: каждый этап = сборка + проверки + commit.

## Этапы

| # | Название | Коммит | Статус |
|---|----------|--------|--------|
| 1 | **Каркас**: конфиг, логирование, трей, иконки, меню. Без аудио. + сквозное логирование/хронометраж | `Этап 1: каркас` | ⏳ |
| 2 | **Захват аудио** через malgo + выбор устройства по имени | `Этап 2: аудио` | ⬜ |
| 3 | **Vosk Grammar Mode**: детекция wake/stop-фраз, стоп-фразы локально | `Этап 3: vosk` | ⬜ |
| 4 | **Облачный STT** (RouterAI Whisper) + WAV-обёртка | `Этап 4: stt` | ⬜ |
| 5 | **AI-роутинг интентов** (LLM-классификатор команд) | `Этап 5: intents` | ⬜ |
| 6 | **Вызов Claude** через RouterAI + сборка вики-промпта из prompts/ | `Этап 6: claude` | ⬜ |
| 7 | **Умная вставка** в активное окно (SendInput) + буфер | `Этап 7: вставка` | ⬜ |
| 8 | **Горячие фразы** (локальная подстановка без LLM) | `Этап 8: горячие фразы` | ⬜ |
| 9 | **Режим «Вася, поправь»** (правка выделенного) | `Этап 9: правка` | ⬜ |
| 10 | **Защита от галлюцинаций** (проверка ключевых терминов) | `Этап 10: антигаллюцинации` | ⬜ |
| 11 | **Голосовые заметки** о недочётах (запись в MD-файл) | `Этап 11: заметки` | ⬜ |
| 12 | **Финальная сборка .exe, README, отчёт, push** в GitHub | `Этап 12: финал` | ⬜ |

## Архитектура

```
CommandRouter (центральный диспетчер)
  ├── IdleHandler        — wake-события, быстрый путь (Vosk)
  ├── DictationHandler   — диктовка → протокол → вставка
  ├── IntentHandler      — AI-роутинг (CommandRecording → LLM → dispatch)
  ├── CorrectionHandler  — «Вася, поправь»
  └── NoteHandler        — голосовые заметки
```

**Двухуровневая система команд:**
- **Уровень 1 (быстрый)**: Vosk Grammar Mode, маркерные слова, <150 мс, без LLM.
- **Уровень 2 (умный)**: поймали только «Вася» → CommandRecording (5 сек) → Whisper → LLM-классификатор интентов.

## Состояния

`Idle / Recording / Processing / Done / CommandRecording / IntentProcessing / NoteRecording / NoteProcessing` (enum + мьютекс).

## Стек

Go 1.24+, deps: `malgo`, `systray`, `clipboard`, `vosk-api/go`, `golang.org/x/sys/windows` (build tags). Кросс-компиляция Linux → Windows через mingw-w64.

## Логирование (сквозное)

- `logs/protocol-assistant.log` — человекочитаемый, ротация 10 MB × 5.
- `logs/events.jsonl` — JSON Lines, ротация 50 MB × 10.
- `logs/sessions/YYYY-MM-DD_HHMMSS.json` — отчёт сессии.
- `scripts/analyze.py` — анализ events.jsonl.
- События: `app_start/stop`, `session_start/end`, `state_change`, `wake_detected`, `audio_chunk_stats`, `stt_request/response`, `llm_request/response`, `intent_request/response`, `hotphrase_substituted`, `hallucination_check`, `smart_paste_attempt/result`, `research_complete`, `correction_cycle`, `note_recorded`, `error`, `config_loaded`, `prompt_built`.

## Замечания по среде

- Go установлен в `~/go-sdk/go` (без root).
- **mingw-w64 НЕ установлен** — нужен sudo-права для `apt install gcc-mingw-w64`. Либо искать альтернативу (проверить наличие пароля root, использовать TDM-GCC через user-space, или докер-образ).
- URL редактирования/финал: частный репозиторий `protocol-voice-assistant` через GITHUB_TOKEN.