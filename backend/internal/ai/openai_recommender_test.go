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
