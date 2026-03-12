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

type RuleOnlyEngine struct{}

func NewRuleOnlyEngine() *RuleOnlyEngine {
	return &RuleOnlyEngine{}
}

func (e *RuleOnlyEngine) Decide(_ context.Context, req DecisionContext) (RecommendationOutput, bool) {
	if req.Input.Constraints.HasDiscomfort || req.Input.Constraints.WeatherRisk == "red" || req.Input.RecoveryStatus == "red" || req.Input.Constraints.HighLoad {
		return RecommendationOutput{
			ShouldRun:           false,
			WorkoutType:         "rest",
			IntensityRange:      "low",
			TargetVolume:        "0",
			SuggestedTimeWindow: "any",
			RiskLevel:           "red",
			HydrationTip:        "注意补水",
			ClothingTip:         "以保暖和舒适为主",
			Explanation:         []string{"触发安全规则，建议休息", "今日不建议高强度训练"},
			AlternativeWorkouts: []AlternativeWorkout{
				{Type: "mobility", Title: "拉伸与灵活性恢复", DurationMin: 20, Intensity: "low"},
			},
		}, true
	}
	if req.Input.Constraints.WeatherRisk == "yellow" || req.Input.RecoveryStatus == "yellow" {
		return RecommendationOutput{
			ShouldRun:           true,
			WorkoutType:         "easy_run",
			IntensityRange:      "低强度",
			TargetVolume:        "20-30 分钟",
			SuggestedTimeWindow: "any",
			RiskLevel:           "yellow",
			HydrationTip:        "适量补水",
			ClothingTip:         "根据体感分层穿着",
			Explanation:         []string{"当前风险可控，建议降强度", "优先恢复性训练"},
			AlternativeWorkouts: []AlternativeWorkout{},
		}, true
	}
	return RecommendationOutput{
		ShouldRun:           true,
		WorkoutType:         "aerobic_run",
		IntensityRange:      "低-中强度",
		TargetVolume:        "30-45 分钟",
		SuggestedTimeWindow: "any",
		RiskLevel:           "green",
		HydrationTip:        "训练前后补水",
		ClothingTip:         "轻量透气",
		Explanation:         []string{"状态稳定，可进行常规训练", "建议保持节奏并关注体感"},
		AlternativeWorkouts: []AlternativeWorkout{},
	}, true
}
