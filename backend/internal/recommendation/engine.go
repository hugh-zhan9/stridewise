package recommendation

import (
	"context"

	"stridewise/backend/internal/ai"
)

type DecisionContext struct {
	Input      ai.RecommendationInput
	WeatherErr error
}

type DecisionEngine interface {
	Decide(ctx context.Context, req DecisionContext) (RecommendationOutput, bool)
}

type AIPrimaryEngine struct {
	recommender ai.Recommender
}

func NewAIPrimaryEngine(recommender ai.Recommender) *AIPrimaryEngine {
	return &AIPrimaryEngine{recommender: recommender}
}

func (e *AIPrimaryEngine) Decide(ctx context.Context, req DecisionContext) (RecommendationOutput, bool) {
	if req.WeatherErr != nil {
		return fallbackOutput(), true
	}
	if e == nil || e.recommender == nil {
		return fallbackOutput(), true
	}
	out, err := e.recommender.Recommend(ctx, req.Input)
	if err != nil {
		return fallbackOutput(), true
	}
	return convertOutput(out), false
}

