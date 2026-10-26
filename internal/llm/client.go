// Package llm — универсальный клиент RouterAI Chat Completions для классификации
// интентов и генерации протокола (Claude). Все запросы через один интерфейс,
// модель выбирается параметром.
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

// Message — сообщение в цепочке чата.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request — тело запроса к /chat/completions (OpenAI-compatible).
type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Response — структура ответа от /chat/completions.
type Response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// Client — HTTP-клиент для RouterAI /chat/completions.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient создаёт клиент RouterAI.
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://routerai.ru/api/v1/"
	}
	if baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Chat выполняет Chat Completions запрос. Возвращает текстовый ответ + мета-инфо.
func (c *Client) Chat(req Request, requestID string) (string, *Info, error) {
	l := logger.Get()

	start := time.Now()

	body, _ := json.Marshal(req)
	l.Event("llm_request", map[string]any{
		"model":             req.Model,
		"prompt_size_bytes": len(body),
		"request_id":        requestID,
		"max_tokens":        req.MaxTokens,
	})

	raw, err := c.doPost("chat/completions", body)
	latency := time.Since(start)

	if err != nil {
		l.Event("llm_response", map[string]any{
			"latency_ms": latency.Milliseconds(),
			"error":      err.Error(),
			"request_id": requestID,
		})
		return "", nil, fmt.Errorf("llm: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		l.Event("llm_response", map[string]any{
			"latency_ms": latency.Milliseconds(),
			"error":      fmt.Sprintf("parse: %v", err),
			"request_id": requestID,
		})
		return "", nil, fmt.Errorf("llm: parse response: %w", err)
	}

	if len(resp.Choices) == 0 {
		l.Event("llm_response", map[string]any{
			"latency_ms": latency.Milliseconds(),
			"error":      "no choices in response",
			"request_id": requestID,
		})
		return "", nil, fmt.Errorf("llm: пустой ответ (no choices)")
	}

	text := resp.Choices[0].Message.Content

	info := &Info{
		OutputTokens: 0,
		OutputLength: len(text),
		LatencyMs:    latency.Milliseconds(),
		StatusCode:   200,
	}
	if resp.Usage != nil {
		info.InputTokens = resp.Usage.PromptTokens
		info.OutputTokens = resp.Usage.CompletionTokens
	}

	l.Event("llm_response", map[string]any{
		"output_tokens": info.OutputTokens,
		"output_length": info.OutputLength,
		"latency_ms":    latency.Milliseconds(),
		"status_code":   200,
		"request_id":    requestID,
	})

	return text, info, nil
}

// Info — метаинформация о запросе.
type Info struct {
	InputTokens  int   `json:"input_tokens"`
	OutputTokens int   `json:"output_tokens"`
	OutputLength int   `json:"output_length"`
	LatencyMs    int64 `json:"latency_ms"`
	StatusCode   int   `json:"status_code"`
}

func (c *Client) doPost(endpoint string, body []byte) ([]byte, error) {
	url := c.baseURL + endpoint
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}
