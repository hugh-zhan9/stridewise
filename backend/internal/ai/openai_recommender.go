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

type OpenAIRecommender struct {
	cfg    OpenAIConfig
	client *http.Client
}

func NewOpenAIRecommender(cfg OpenAIConfig) *OpenAIRecommender {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if cfg.TimeoutMs <= 0 {
		timeout = 3 * time.Second
	}
	return &OpenAIRecommender{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (r *OpenAIRecommender) Recommend(ctx context.Context, input RecommendationInput) (RecommendationOutput, error) {
	if r.cfg.APIKey == "" {
		return RecommendationOutput{}, errors.New("openai api_key required")
	}
	if r.cfg.Model == "" {
		return RecommendationOutput{}, errors.New("openai model required")
	}
	baseURL := strings.TrimRight(r.cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := baseURL + "/chat/completions"

	payload, err := buildOpenAIRecRequest(r.cfg, input)
	if err != nil {
		return RecommendationOutput{}, err
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return RecommendationOutput{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return RecommendationOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return RecommendationOutput{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RecommendationOutput{}, err
	}
	if resp.StatusCode >= 300 {
		return RecommendationOutput{}, fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return RecommendationOutput{}, err
	}
	if len(parsed.Choices) == 0 {
		return RecommendationOutput{}, errors.New("openai empty choices")
	}
	content := parsed.Choices[0].Message.Content
	if content == "" {
		return RecommendationOutput{}, errors.New("openai empty content")
	}
	var out RecommendationOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return RecommendationOutput{}, err
	}
	if err := validateRecommendationOutput(out); err != nil {
		return RecommendationOutput{}, err
	}
	return out, nil
}

func buildOpenAIRecRequest(cfg OpenAIConfig, input RecommendationInput) (openAIRequest, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return openAIRequest{}, err
	}
	systemPrompt := "你是跑步训练建议助手。请生成结构化建议，确保安全优先。"
	userPrompt := fmt.Sprintf("请基于以下JSON生成训练建议，仅输出JSON对象，字段必须包含 should_run、workout_type、intensity_range、target_volume、suggested_time_window、risk_level、hydration_tip、clothing_tip、explanation。输入JSON: %s", string(inputJSON))
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

func validateRecommendationOutput(out RecommendationOutput) error {
	if out.WorkoutType == "" {
		return errors.New("workout_type required")
	}
	if out.IntensityRange == "" {
		return errors.New("intensity_range required")
	}
	if out.TargetVolume == "" {
		return errors.New("target_volume required")
	}
	if out.SuggestedTimeWindow == "" {
		return errors.New("suggested_time_window required")
	}
	if out.RiskLevel == "" {
		return errors.New("risk_level required")
	}
	if len(out.Explanation) < 2 {
		return errors.New("explanation requires at least 2 items")
	}
	return nil
}
