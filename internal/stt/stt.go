// Package stt — облачное распознавание речи через RouterAI Whisper.
// Отправляет WAV-файл (16 кГц, моно, S16LE) на /chat/completions с аудио-моделью
// (например, openai/whisper-large-v3) и получает транскрипцию.
package stt

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// Client — STT-клиент через RouterAI.
type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
}

// NewClient создаёт STT-клиент.
func NewClient(apiKey, baseURL, model string) *Client {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// TranscribePCM принимает []int16 (S16LE, моно, 16000 Гц), оборачивает в WAV
// и отправляет на RouterAI Whisper. Возвращает распознанный текст.
func (c *Client) TranscribePCM(samples []int16, opts Options) (string, error) {
	if len(samples) == 0 {
		return "", fmt.Errorf("stt: пустой аудио-буфер")
	}

	wav, err := encodeWAV(samples)
	if err != nil {
		return "", fmt.Errorf("stt: wav encode: %w", err)
	}

	return c.transcribeWAV(wav, opts)
}

// Options — параметры запроса STT.
type Options struct {
	Language string // пусто = авто
	// RequestID для связывания логов (опционально)
	RequestID string
}

// transcribeWAV отправляет WAV-файл на RouterAI Whisper.
func (c *Client) transcribeWAV(wavData []byte, opts Options) (string, error) {
	l := logger.Get()
	start := time.Now()

	lgReq := logger.Get()
	lgReq.Event("stt_request", map[string]any{
		"audio_duration_ms": len(wavData) / 16000 / 2 * 1000, // грубая оценка
		"audio_size_bytes":  len(wavData),
		"model":             c.model,
		"endpoint":          c.baseURL + "audio/transcriptions",
		"request_id":        opts.RequestID,
	})

	// Формируем multipart/form-data для RouterAI (OpenAI-compatible).
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// файл аудио
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("stt: create form file: %w", err)
	}
	if _, err := part.Write(wavData); err != nil {
		return "", fmt.Errorf("stt: write wav: %w", err)
	}
	// модель
	if err := writer.WriteField("model", c.model); err != nil {
		return "", fmt.Errorf("stt: write model field: %w", err)
	}
	// язык (опционально)
	if opts.Language != "" {
		_ = writer.WriteField("language", opts.Language)
	}
	// response_format = json
	_ = writer.WriteField("response_format", "json")
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("stt: close writer: %w", err)
	}

	url := c.baseURL + "audio/transcriptions"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("stt: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		l.Error("stt: request failed: " + err.Error())
		_ = lgReq
		return "", fmt.Errorf("stt: request: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		lgReq.Event("stt_response", map[string]any{
			"latency_ms":  latency.Milliseconds(),
			"status_code": resp.StatusCode,
			"error":       fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			"request_id":  opts.RequestID,
		})
		return "", fmt.Errorf("stt: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Парсим ответ. RouterAI возвращает OpenAI-compatible JSON.
	// {"text": "..."}
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		lgReq.Event("stt_response", map[string]any{
			"latency_ms":  latency.Milliseconds(),
			"status_code": resp.StatusCode,
			"error":       fmt.Sprintf("parse response: %v", err),
			"request_id":  opts.RequestID,
		})
		return "", fmt.Errorf("stt: parse response: %w", err)
	}

	lgReq.Event("stt_response", map[string]any{
		"text_length": len(result.Text),
		"latency_ms":  latency.Milliseconds(),
		"status_code": resp.StatusCode,
		"request_id":  opts.RequestID,
	})
	l.Info(fmt.Sprintf("stt: распознано %d символов за %v", len(result.Text), latency.Round(time.Millisecond)))

	return result.Text, nil
}

// encodeWAV оборачивает []int16 S16LE моно в WAV-контейнер (16000 Гц).
func encodeWAV(samples []int16) ([]byte, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("stt: no samples")
	}

	numSamples := len(samples)
	numChannels := uint16(1)
	sampleRate := uint32(16000)
	bitsPerSample := uint16(16)
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample/8)
	blockAlign := numChannels * (bitsPerSample / 8)
	dataSize := uint32(numSamples) * uint32(blockAlign)
	fileSize := 36 + dataSize

	buf := new(bytes.Buffer)

	// RIFF-заголовок
	writeStr(buf, "RIFF")
	binary.Write(buf, binary.LittleEndian, fileSize)
	writeStr(buf, "WAVE")

	// fmt-чанк
	writeStr(buf, "fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))  // chunk size
	binary.Write(buf, binary.LittleEndian, uint16(1))   // PCM
	binary.Write(buf, binary.LittleEndian, numChannels) // mono
	binary.Write(buf, binary.LittleEndian, sampleRate)  // 16000
	binary.Write(buf, binary.LittleEndian, byteRate)
	binary.Write(buf, binary.LittleEndian, blockAlign)
	binary.Write(buf, binary.LittleEndian, bitsPerSample)

	// data-чанк
	writeStr(buf, "data")
	binary.Write(buf, binary.LittleEndian, dataSize)
	for _, s := range samples {
		binary.Write(buf, binary.LittleEndian, s)
	}

	return buf.Bytes(), nil
}

func writeStr(buf *bytes.Buffer, s string) {
	buf.WriteString(s)
}
