package ai

import (
	"context"
	"testing"
)

func TestOpenAIRecommenderRequiresConfig(t *testing.T) {
	r := NewOpenAIRecommender(OpenAIConfig{})
	_, err := r.Recommend(context.Background(), RecommendationInput{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateRecommendationOutputAllowsSingleExplanation(t *testing.T) {
	output := RecommendationOutput{
		ShouldRun:           true,
		WorkoutType:         "easy_run",
		IntensityRange:      "low",
		TargetVolume:        "3 km",
		SuggestedTimeWindow: "any",
		RiskLevel:           "green",
		HydrationTip:        "",
		ClothingTip:         "",
		Explanation:         []string{"天气良好"},
	}

	if err := validateRecommendationOutput(output); err != nil {
		t.Fatalf("expected single explanation accepted, got error: %v", err)
	}
}
