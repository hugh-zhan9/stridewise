package recommendation

import "testing"

func TestApplyRules_RedWeatherOverrides(t *testing.T) {
	input := RuleInput{WeatherRisk: "red"}
	out := RecommendationOutput{ShouldRun: true}
	result := ApplyRules(input, out)
	if result.Output.ShouldRun {
		t.Fatalf("expected override to rest")
	}
	if result.OverrideReason == "" {
		t.Fatalf("expected override reason")
	}
}

func TestApplyRules_AddsAlternativeWorkoutsOnOverride(t *testing.T) {
	input := RuleInput{WeatherRisk: "red"}
	out := RecommendationOutput{ShouldRun: true}
	result := ApplyRules(input, out)
	if len(result.Output.AlternativeWorkouts) == 0 {
		t.Fatalf("expected alternative workouts")
	}
}
