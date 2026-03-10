package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	TimeoutMs   int
	MaxTokens   int
	Temperature float64
}

type OpenAISummarizer struct {
	cfg    OpenAIConfig
	client *http.Client
}

func NewOpenAISummarizer(cfg OpenAIConfig) *OpenAISummarizer {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if cfg.TimeoutMs <= 0 {
		timeout = 3 * time.Second
	}
	return &OpenAISummarizer{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *OpenAISummarizer) Summarize(ctx context.Context, input SummaryInput) (SummaryOutput, error) {
	if s.cfg.APIKey == "" {
		return SummaryOutput{}, errors.New("openai api_key required")
	}
	if s.cfg.Model == "" {
		return SummaryOutput{}, errors.New("openai model required")
	}
	baseURL := strings.TrimRight(s.cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := baseURL + "/chat/completions"

	payload, err := buildOpenAIRequest(s.cfg, input)
	if err != nil {
		return SummaryOutput{}, err
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return SummaryOutput{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return SummaryOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return SummaryOutput{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SummaryOutput{}, err
	}
	if resp.StatusCode >= 300 {
		return SummaryOutput{}, fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return SummaryOutput{}, err
	}
	if len(parsed.Choices) == 0 {
		return SummaryOutput{}, errors.New("openai empty choices")
	}
	content := parsed.Choices[0].Message.Content
	if content == "" {
		return SummaryOutput{}, errors.New("openai empty content")
	}

	var out SummaryOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return SummaryOutput{}, err
	}
	if err := validateSummaryOutput(out); err != nil {
		return SummaryOutput{}, err
	}
	return out, nil
}

type openAIRequest struct {
	Model          string            `json:"model"`
	Messages       []openAIMessage   `json:"messages"`
	ResponseFormat *openAIRespFormat `json:"response_format,omitempty"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Temperature    float64           `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRespFormat struct {
	Type string `json:"type"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func buildOpenAIRequest(cfg OpenAIConfig, input SummaryInput) (openAIRequest, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return openAIRequest{}, err
	}
	systemPrompt := "你是跑步训练总结助手。请根据训练记录与基线指标生成训练后总结。"
	userPrompt := fmt.Sprintf("请基于以下JSON生成训练总结，仅输出JSON对象，字段必须包含 completion_rate、intensity_match、recovery_advice、anomaly_notes、performance_notes、next_suggestion。输入JSON: %s", string(inputJSON))
	req := openAIRequest{
		Model: cfg.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &openAIRespFormat{Type: "json_object"},
	}
	if cfg.MaxTokens > 0 {
		req.MaxTokens = cfg.MaxTokens
	}
	if cfg.Temperature > 0 {
		req.Temperature = cfg.Temperature
	}
	return req, nil
}

func validateSummaryOutput(out SummaryOutput) error {
	if strings.TrimSpace(out.CompletionRate) == "" {
		return errors.New("completion_rate required")
	}
	if strings.TrimSpace(out.IntensityMatch) == "" {
		return errors.New("intensity_match required")
	}
	if strings.TrimSpace(out.RecoveryAdvice) == "" {
		return errors.New("recovery_advice required")
	}
	if strings.TrimSpace(out.AnomalyNotes) == "" {
		return errors.New("anomaly_notes required")
	}
	if strings.TrimSpace(out.PerformanceNotes) == "" {
		return errors.New("performance_notes required")
	}
	if strings.TrimSpace(out.NextSuggestion) == "" {
		return errors.New("next_suggestion required")
	}
	return nil
}
