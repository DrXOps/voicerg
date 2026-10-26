// Package prompt — сборка вики-промпта для Claude из набора markdown-файлов.
// Поддерживает ссылки вида [[file.md]] внутри контента: содержимое такого файла
// рекурсивно встраивается. Защита от циклов и от каталог-злоупотреблений.
package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// wikiRef — регулярка для ссылок [[name]].
var wikiRef = regexp.MustCompile(`\[\[([^\]\[]+)\]\]`)

// WikiBuilder собирает единый системный промпт из файлов вики.
type WikiBuilder struct {
	dir string
}

// NewWikiBuilder создаёт сборщик вики из каталога dir (prompts/).
func NewWikiBuilder(dir string) *WikiBuilder {
	return &WikiBuilder{dir: dir}
}

// Build собирает промпт, начиная с mainFile (например, "_index.md").
// Возвращает: итоговый текст, список включённых файлов, информацию о сборке.
func (w *WikiBuilder) Build(mainFile string) (string, BuildInfo, error) {
	l := logger.Get()
	start := time.Now()

	var info BuildInfo
	info.MainFile = mainFile
	visited := make(map[string]bool)
	var sb strings.Builder

	content, err := w.resolve(filepath.Clean(mainFile), visited, &info, 0)
	if err != nil {
		return "", info, fmt.Errorf("prompt: build wiki: %w", err)
	}
	sb.WriteString(content)

	info.TotalSizeBytes = sb.Len()
	info.BuildDurationMs = time.Since(start).Milliseconds()

	l.Event("prompt_built", map[string]any{
		"main_file":           mainFile,
		"files_included":      info.FilesIncluded,
		"total_size_bytes":    info.TotalSizeBytes,
		"build_duration_ms":   info.BuildDurationMs,
		"recursion_depth_max": info.RecursionDepthMax,
	})

	return sb.String(), info, nil
}

// BuildInfo — метаданные о сборке промпта.
type BuildInfo struct {
	MainFile          string     `json:"main_file"`
	FilesIncluded     []FileInfo `json:"files_included"`
	TotalSizeBytes    int        `json:"total_size_bytes"`
	BuildDurationMs   int64      `json:"build_duration_ms"`
	RecursionDepthMax int        `json:"recursion_depth_max"`
	MissingLinks      []string   `json:"missing_links"`
}

// FileInfo — информация о включённом файле.
type FileInfo struct {
	Name      string `json:"name"`
	SizeBytes int    `json:"size_bytes"`
}

// resolve рекурсивно обрабатывает файл, встраивая [[ссылки]].
func (w *WikiBuilder) resolve(relPath string, visited map[string]bool, info *BuildInfo, depth int) (string, error) {
	l := logger.Get()

	if depth > info.RecursionDepthMax {
		info.RecursionDepthMax = depth
	}

	// защита от циклов
	if visited[relPath] {
		l.Warn("prompt: цикл в вики, пропускаю " + relPath)
		return "", nil
	}
	// защита от выхода за каталог
	if strings.Contains(relPath, "..") {
		l.Warn("prompt: недопустимый путь в вики: " + relPath)
		return "", fmt.Errorf("prompt: unsafe path %q", relPath)
	}

	visited[relPath] = true

	abs := filepath.Join(w.dir, relPath)
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			info.MissingLinks = append(info.MissingLinks, relPath)
			l.Warn("prompt: файл не найден, оставляю ссылку: " + relPath)
			return "[[**ФАЙЛ НЕ НАЙДЕН: " + relPath + "**]]", nil
		}
		return "", fmt.Errorf("prompt: read %s: %w", relPath, err)
	}

	content := string(data)
	info.FilesIncluded = append(info.FilesIncluded, FileInfo{
		Name:      relPath,
		SizeBytes: len(data),
	})

	// Встраиваем [[ссылки]] рекурсивно.
	base := filepath.Dir(relPath)
	content = wikiRef.ReplaceAllStringFunc(content, func(m string) string {
		name := m[2 : len(m)-2] // убираем [[ ]]
		name = strings.TrimSpace(name)
		// резолвим относительно текущей папки
		child := name
		if base != "." && base != "" && !filepath.IsAbs(child) {
			child = filepath.Join(base, child)
		}
		child = filepath.Clean(child)
		if visited[child] {
			l.Warn("prompt: цикл-ссылка: " + child)
			return m // оставляем как есть
		}
		sub, err := w.resolve(child, visited, info, depth+1)
		if err != nil {
			l.Warn("prompt: ошибка вложения " + child + ": " + err.Error())
			return m
		}
		return "\n<!-- Вики-файл: " + name + " -->\n" + sub + "\n"
	})

	return content, nil
}
