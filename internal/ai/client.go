package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"diagnostic-studio/internal/model"
)

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func (s Store) Explain(ctx context.Context, r model.DiagnosticReport, lang string) model.AIExplanation {
	cfg, err := s.Load()
	if err != nil {
		return failed(cfg.Model, "load AI config: "+err.Error())
	}
	if !cfg.Enabled {
		return model.AIExplanation{Status: "disabled", Model: cfg.Model}
	}
	if !cfg.HasAPIKey {
		return model.AIExplanation{Status: "missing-key", Model: cfg.Model}
	}
	key, err := s.apiKey()
	if err != nil {
		return failed(cfg.Model, "decrypt API key: "+err.Error())
	}
	if key == "" {
		return model.AIExplanation{Status: "missing-key", Model: cfg.Model}
	}
	summary, err := RedactedSummary(r)
	if err != nil {
		return failed(cfg.Model, "build summary: "+err.Error())
	}

	reqBody := chatRequest{
		Model:       cfg.Model,
		Temperature: 0.2,
		MaxTokens:   900,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt(lang)},
			{Role: "user", Content: "Reduced diagnostic JSON:\n" + summary},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return failed(cfg.Model, "marshal request: "+err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsURL(cfg.BaseURL), bytes.NewReader(body))
	if err != nil {
		return failed(cfg.Model, "create request: "+err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return failed(cfg.Model, "call AI endpoint: "+err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return failed(cfg.Model, "read response: "+err.Error())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return failed(cfg.Model, fmt.Sprintf("endpoint returned HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 500)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return failed(cfg.Model, "parse response: "+err.Error())
	}
	if parsed.Error != nil {
		return failed(cfg.Model, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return failed(cfg.Model, "empty AI response")
	}
	return model.AIExplanation{
		Status:  "success",
		Content: strings.TrimSpace(parsed.Choices[0].Message.Content),
		Model:   cfg.Model,
		SentAt:  time.Now().Format(time.RFC3339),
	}
}

func chatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func systemPrompt(lang string) string {
	language := "English"
	if lang == "zh" {
		language = "Simplified Chinese"
	}
	return "You are the optional AI explanation layer for Diagnostic Studio, a Windows PC diagnostic tool. " +
		"Use only the supplied reduced diagnostic JSON. Do not invent causes, measurements, or actions. " +
		"Local rules are authoritative; you may explain and prioritize but must never claim you executed anything. " +
		"If evidence is inconclusive, say so plainly. Avoid recommending service or startup disabling unless the supplied actions already include it. " +
		"Respond in " + language + " with concise sections: likely meaning, key evidence, next steps, cautions."
}

func failed(modelName, detail string) model.AIExplanation {
	return model.AIExplanation{Status: "failed", Detail: detail, Model: modelName, SentAt: time.Now().Format(time.RFC3339)}
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
