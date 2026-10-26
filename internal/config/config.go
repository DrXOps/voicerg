package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config — корневая конфигурация приложения. Пути относительные — относительно
// каталога рядом с exe (WorkingDir).
type Config struct {
	// Аудио устройство (подстрока имени). Пусто = системное по умолчанию.
	DeviceName string `json:"device_name"`

	// Директория промптов (вики).
	PromptsDir string `json:"prompts_dir"`
	// Главный файл вики-промпта.
	MainPromptFile string `json:"main_prompt_file"`

	// Списки фраз (регистронезависимы, нижний регистр при загрузке).
	WakeWords     []string `json:"wake_words"`
	StopWords     []string `json:"stop_words"`
	NoteWakeWords []string `json:"note_wake_words"`
	NoteStopWords []string `json:"note_stop_words"`

	// Маркерные слова для двухуровневой системы.
	MarkerWords MarkerWords `json:"marker_words"`

	// Команды вставки блоков: ключ — распознанная фраза, значение — маркер блока.
	InsertCommands map[string]string `json:"insert_commands"`

	// Триггеры профилей исследования: слово -> файл шаблона.
	ProfileTriggers map[string]string `json:"profile_triggers"`

	VoskModelPath     string `json:"vosk_model_path"`
	HotphrasesPath    string `json:"hotphrases_path"`
	CriticalTermsPath string `json:"critical_terms_path"`

	// Умная вставка в активное окно (Ctrl+V). false = только в буфер.
	SmartPaste bool `json:"smart_paste"`

	NotesFile string `json:"notes_file"`

	// Промпты, загружаемые конкретными хендлерами (НЕ часть вики).
	NotePromptFile   string `json:"note_prompt_file"`
	IntentPromptFile string `json:"intent_prompt_file"`

	// AI-роутинг.
	IntentModel               string  `json:"intent_model"`
	IntentConfidenceThreshold float64 `json:"intent_confidence_threshold"`

	RouterAIKey     string `json:"routerai_key"`
	RouterAIBaseURL string `json:"routerai_base_url"`

	STTModel    string `json:"stt_model"`
	ClaudeModel string `json:"claude_model"`

	// Насчитанное ниже — не из конфига, а из окружения/файла.
	workingDir string
	hash       string
}

type MarkerWords struct {
	WakeMarkers []string `json:"wake_markers"`
	StopMarkers []string `json:"stop_markers"`
}

// Default возвращает конфигурацию по умолчанию — используется при отсутствии
// config.json (автогенерация, чтобы приложение стартовало с первого раза).
func Default() *Config {
	return &Config{
		DeviceName:     "Webcam",
		PromptsDir:     "prompts",
		MainPromptFile: "_index.md",
		WakeWords: []string{
			"вася", "вася пишем", "вася пиши", "вася начинай", "вася слушай", "вася давай",
			"пишем", "начинаем", "записывай",
			"так пишем", "так начинаем", "ну что пишем",
			"опа поехали", "опа пишем", "опа понеслась",
			"так коленочка пишем", "так колено пишем",
			"так огэкашечка пишем", "опа огэкашечка поехали",
			"так позвоночник пишем", "так спина пишем",
			"так стопа пишем", "так кисть пишем",
			"коленочка поехали", "огэкашечка поехали",
			"поехали писать",
		},
		StopWords: []string{
			"заключение", "закончил", "готово", "всё", "вася стоп",
			"конец протокола", "хватит", "баста",
		},
		NoteWakeWords: []string{
			"вася заметка", "вася баг", "вася проблема", "вася идея",
			"вася зафиксируй", "вася ошибка", "вася глюк", "вася это не работает",
			"зафиксируй",
		},
		NoteStopWords: []string{
			"конец заметки", "закончил заметку", "всё по заметке",
		},
		MarkerWords: MarkerWords{
			WakeMarkers: []string{
				"вася", "пишем", "начинаем", "поехали", "погнали", "стартуем",
				"понеслись", "работаем", "слушай", "давай",
			},
			StopMarkers: []string{
				"заключение", "закончил", "готово", "всё", "стоп",
				"хватит", "конец", "баста",
			},
		},
		InsertCommands: map[string]string{
			"заключение":   "ЗАКЛЮЧЕНИЕ",
			"описание":     "ОПИСАНИЕ",
			"рекомендации": "РЕКОМЕНДАЦИИ",
		},
		ProfileTriggers: map[string]string{
			"колено":         "knee.md",
			"коленочка":      "knee.md",
			"огк":            "chest.md",
			"огэкашечка":     "chest.md",
			"грудная клетка": "chest.md",
			"позвоночник":    "spine.md",
			"спина":          "spine.md",
			"стопа":          "foot.md",
			"стопочка":       "foot.md",
			"кисть":          "hand.md",
			"кисточка":       "hand.md",
		},
		VoskModelPath:     "models/vosk-model-small-ru-0.22",
		HotphrasesPath:    "hotphrases.json",
		CriticalTermsPath: "critical_terms.json",
		SmartPaste:        true,
		NotesFile:         "notes/feedback.md",
		NotePromptFile:    "prompts/note_prompt.md",
		IntentPromptFile:  "prompts/intent_prompt.md",
		// Врач может поменять на любую доступную через RouterAI, например:
		// "anthropic/claude-haiku-latest" или "google/gemini-flash-latest".
		IntentModel:               "openai/gpt-4o-mini",
		IntentConfidenceThreshold: 0.6,
		RouterAIKey:              "sk-aoiVlIf819ZjzWceXbHDU97fI_SWQppI",
		RouterAIBaseURL:           "https://routerai.ru/api/v1",
		STTModel:                  "openai/whisper-large-v3",
		ClaudeModel:               "anthropic/claude-sonnet-5.5",
	}
}

// WorkingDir — каталог, рядом с exe. Все относительные пути резолвятся от него.
func (c *Config) WorkingDir() string { return c.workingDir }

// Hash — SHA256 первых 1 КБ config.json (для событий аналитики).
func (c *Config) Hash() string { return c.hash }

// Load читает config.json из dir (обычно рядом с exe). Если файла нет —
// создаёт config.json с конфигурацией по умолчанию и логирует. Все строки
// нормализуются в нижний регистр.
func Load(dir string) (*Config, error) {
	c := Default()
	c.workingDir = dir

	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Нет конфига — записываем образец, стартуем с дефолтом.
			cfgBytes, _ := json.MarshalIndent(c, "", "  ")
			_ = os.WriteFile(path, cfgBytes, 0o644)
			c.normalize()
			c.computeHash(cfgBytes)
			return c, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	c2 := Default()
	if err := json.Unmarshal(data, c2); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	c2.workingDir = dir
	c2.normalize()
	c2.computeHash(data)
	return c2, nil
}

// Resolve возвращает абсолютный путь к p: если p относительный — резолвит от
// каталога рядом с exe.
func (c *Config) Resolve(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.workingDir, p)
}

// normalize переводит все пользовательские фразы в нижний регистр
// (регистронезависимость детекции).
func (c *Config) normalize() {
	for i := range c.WakeWords {
		c.WakeWords[i] = strLower(c.WakeWords[i])
	}
	for i := range c.StopWords {
		c.StopWords[i] = strLower(c.StopWords[i])
	}
	for i := range c.NoteWakeWords {
		c.NoteWakeWords[i] = strLower(c.NoteWakeWords[i])
	}
	for i := range c.NoteStopWords {
		c.NoteStopWords[i] = strLower(c.NoteStopWords[i])
	}
	for i := range c.MarkerWords.WakeMarkers {
		c.MarkerWords.WakeMarkers[i] = strLower(c.MarkerWords.WakeMarkers[i])
	}
	for i := range c.MarkerWords.StopMarkers {
		c.MarkerWords.StopMarkers[i] = strLower(c.MarkerWords.StopMarkers[i])
	}
}

func (c *Config) computeHash(data []byte) {
	b := make([]byte, 0)
	if len(data) > 1024 {
		b = data[:1024]
	} else {
		b = data
	}
	sum := sha256.Sum256(b)
	c.hash = hex.EncodeToString(sum[:])
}

func strLower(s string) string {
	// strings.ToLower — Unicode-aware (важно для кириллицы в русских фразах).
	return strings.ToLower(s)
}
