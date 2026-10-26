# Вася — голосовой ассистент рентгенолога

Фоновое Windows-приложение в системном трее для голосовой диктовки протоколов
рентгенологических исследований.

## Возможности

- 🎤 Vosk Grammar Mode — быстрый (<150 мс) локальный детектор wake/stop-фраз
- 🧠 Двухуровневая система команд: Vosk (быстрая) + LLM (умная)
- 🗣️ Облачный STT (RouterAI Whisper) — расшифровка речи
- 📝 Claude (через RouterAI) — генерация структурированного протокола
- 🔧 Локальная подстановка горячих фраз (без обращения к LLM)
- 📋 Умная вставка в активное окно МИС (Ctrl+V через SendInput)
- ✏️ Режим «Вася, поправь» — правка выделенного текста через LLM
- 🛡️ Защита от галлюцинаций — проверка критических терминов
- 📝 Голосовые заметки о недочётах (feedback.md для анализа)
- 📊 Сквозной хронометраж: events.jsonl + analyze.py

## Быстрый старт (Windows)

1. **Установите .exe:** скачайте `protocol-voice-assistant.exe` в любую папку.
2. **Положите рядом vosk.dll** ([скачать](https://github.com/alphacep/vosk-api/releases))
   и модель `models/vosk-model-small-ru-0.22/`
   ([скачать](https://alphacephei.com/vosk/models/vosk-model-small-ru-0.22.zip)).
3. **Настройте API-ключ RouterAI:** отредактируйте `config.json` (создаётся
   автоматически при первом запуске):
   ```json
   "routerai_key": "sk-your-key-here"
   ```
4. **Запустите.** Иконка появится в системном трее.
5. **Произнесите активационную фразу:** «Вася, начинаем» или «Поехали писать».
6. **Продиктуйте протокол.** Скажите «Заключение» для остановки.
7. **Результат** — структурированный протокол в активном окне МИС.

> **Важно:** при первом запуске создаётся `config.json` с конфигурацией по умолчанию.
> Отредактируйте `routerai_key`, `intent_model` (можно сменить на
> `anthropic/claude-haiku-latest` или `google/gemini-flash-latest`),
> `stt_model` и `claude_model` под свои нужды.

## Список файлов проекта

```
├── cmd/protocol-assistant/   — точка входа (main)
│   ├── main.go              — инициализация и запуск
│   └── audio_monitor.go     — мониторинг аудио-чанков
├── internal/
│   ├── audio/               — захват аудио (malgo, VAD)
│   ├── config/              — загрузка config.json
│   ├── correction/          — режим «Вася, поправь»
│   ├── engine/              — CommandRouter + хендлеры
│   ├── hallucination/       — защита от галлюцинаций
│   ├── hotphrases/          — локальная подстановка фраз
│   ├── icons/               — программная генерация .ico
│   ├── llm/                 — RouterAI Chat Completions + интенты
│   ├── logger/              — двухфайловое логирование + хронометраж
│   ├── note/                — голосовые заметки (feedback.md)
│   ├── paste/               — умная вставка (SendInput, буфер)
│   ├── prompt/              — вики-сборщик промптов
│   ├── protocol/            — генератор протокола (BlockParser)
│   ├── state/               — конечный автомат состояний
│   ├── stt/                 — облачный STT (RouterAI Whisper + WAV)
│   └── wake/                — Vosk Grammar Mode
├── prompts/                 — вики-промпты (_index, chest, knee, …)
├── notes/                   — голосовые заметки (feedback.md)
├── scripts/
│   └── analyze.py           — генератор отчёта из events.jsonl
├── PLan.md                  — пошаговый план этапов
├── Makefile                 — сборка Linux/Windows (Docker)
└── Dockerfile.win           — образ для cross-компиляции (mingw)
```

## Работа с заметками

Голосовые заметки сохраняются в `notes/feedback.md`. Чтобы проанализировать
их через Claude Code:

```bash
cat notes/feedback.md | claude -p "Проанализируй заметки о недочётах ассистента Вася, сгруппируй по категориям, предложи приоритеты исправлений."
```

Для объективного анализа также используйте events.jsonl:

```bash
python3 scripts/analyze.py --events=logs/events.jsonl --since=7d
```

## Конфигурация

Файл `config.json` (автогенерация при первом запуске). Ключевые поля:

| Поле | Описание | По умолчанию |
|------|----------|-------------|
| `device_name` | подстрока имени аудиоустройства | "Webcam" |
| `routerai_key` | **обязательный** API-ключ RouterAI | "sk-..." |
| `intent_model` | модель классификации интентов | "openai/gpt-4o-mini" |
| `stt_model` | модель распознавания речи | "openai/whisper-large-v3" |
| `claude_model` | модель генерации протокола | "anthropic/claude-sonnet-5.5" |
| `smart_paste` | умная вставка в активное окно | true |
| `vosk_model_path` | путь к модели Vosk | "models/vosk-model-small-ru-0.22" |
| `intent_confidence_threshold` | порог автоисполнения интента | 0.6 |

## Сборка

### Linux (проверка логики):

```bash
make build-linux
```

### Windows (через Docker):

```bash
make docker-image    # один раз
make build-windows-docker
```

Готовый .exe в `bin/protocol-voice-assistant.exe`.

## Тестирование

```bash
make test
# или
go test -count=1 ./...
```

## Типичные проблемы

1. **Пустой трей (нет иконки)** — положите vosk.dll рядом с .exe, проверьте
   config.json.
2. **Intent не срабатывает** — проверьте `intent_model` (доступность через RouterAI)
   и `intent_confidence_threshold` (0.6). В log-файле ищите `intent_response`.
3. **Не вставляет текст** — `smart_paste: false` → только буфер. Смените
   окно → защита от промаха.
4. **Нет звука** — проверьте `device_name` (вызовет `ListDevices` при старте,
   вы увидите список в логе).

## Лицензия

Иконки генерируются программно (stdlib), никакие внешние файлы не требуются.

### Этапы разработки

1. [Каркас: конфиг, логирование, трей, иконки](https://github.com/DrXOps/voicerg/commit/3c87c38)
2. [Аудио: malgo + VAD](https://github.com/DrXOps/voicerg/commit/18aa1fc)
3. [Vosk Grammar Mode](https://github.com/DrXOps/voicerg/commit/68271d4)
4. [Облачный STT (RouterAI Whisper)](https://github.com/DrXOps/voicerg/commit/26a4ae1)
5. [AI-роутинг интентов](https://github.com/DrXOps/voicerg/commit/91d0c7c)
6. [Вызов Claude + вики-промпт](https://github.com/DrXOps/voicerg/commit/8aaedeb)
7. [Умная вставка (SendInput)](https://github.com/DrXOps/voicerg/commit/97157ce)
8. [Горячие фразы](https://github.com/DrXOps/voicerg/commit/233652c)
9. [Режим «Вася, поправь»](https://github.com/DrXOps/voicerg/commit/5489efd)
10. [Защита от галлюцинаций](https://github.com/DrXOps/voicerg/commit/713b21f)
11. [Голосовые заметки](https://github.com/DrXOps/voicerg/commit/390947b)
12. [Финальная сборка, README, push](https://github.com/DrXOps/voicerg/commit/latest)