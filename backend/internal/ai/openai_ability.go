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

type OpenAIAbilityLeveler struct {
	cfg    OpenAIConfig
	client *http.Client
}

func NewOpenAIAbilityLeveler(cfg OpenAIConfig) *OpenAIAbilityLeveler {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if cfg.TimeoutMs <= 0 {
		timeout = 3 * time.Second
	}
	return &OpenAIAbilityLeveler{
		cfg: cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (s *OpenAIAbilityLeveler) EvaluateAbilityLevel(ctx context.Context, input AbilityLevelInput) (AbilityLevelOutput, error) {
	if s.cfg.APIKey == "" {
		return AbilityLevelOutput{}, errors.New("openai api_key required")
	}
	if s.cfg.Model == "" {
		return AbilityLevelOutput{}, errors.New("openai model required")
	}
	baseURL := strings.TrimRight(s.cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := baseURL + "/chat/completions"

	payload, err := buildOpenAIAbilityRequest(s.cfg, input)
	if err != nil {
		return AbilityLevelOutput{}, err
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return AbilityLevelOutput{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return AbilityLevelOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return AbilityLevelOutput{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AbilityLevelOutput{}, err
	}
	if resp.StatusCode >= 300 {
		return AbilityLevelOutput{}, fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return AbilityLevelOutput{}, err
	}
	if len(parsed.Choices) == 0 {
		return AbilityLevelOutput{}, errors.New("openai empty choices")
	}
	content := parsed.Choices[0].Message.Content
	if content == "" {
		return AbilityLevelOutput{}, errors.New("openai empty content")
	}

	var out AbilityLevelOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return AbilityLevelOutput{}, err
	}
	if err := validateAbilityLevelOutput(out); err != nil {
		return AbilityLevelOutput{}, err
	}
	return out, nil
}

func buildOpenAIAbilityRequest(cfg OpenAIConfig, input AbilityLevelInput) (openAIRequest, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return openAIRequest{}, err
	}
	systemPrompt := "你是跑步能力分级助手。请根据训练摘要与问卷数据判断能力层级。"
	userPrompt := fmt.Sprintf("请基于以下JSON输出能力层级，仅输出JSON对象，字段必须包含 ability_level（beginner/intermediate/advanced）与可选 reason。输入JSON: %s", string(inputJSON))
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

func validateAbilityLevelOutput(out AbilityLevelOutput) error {
	switch strings.TrimSpace(out.AbilityLevel) {
	case "beginner", "intermediate", "advanced":
		return nil
	default:
		return errors.New("ability_level invalid")
	}
}
