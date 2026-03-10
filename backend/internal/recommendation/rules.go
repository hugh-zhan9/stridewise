package recommendation

type RuleInput struct {
	WeatherRisk   string
	HasDiscomfort bool
	HighLoad      bool
}

type RuleResult struct {
	Output         RecommendationOutput
	OverrideReason string
}

func ApplyRules(input RuleInput, output RecommendationOutput) RuleResult {
	override := ""
	if input.WeatherRisk == "red" {
		override = "weather_red"
	}
	if input.HasDiscomfort {
		override = "user_discomfort"
	}
	if input.HighLoad {
		override = "high_load"
	}
	if override != "" {
		output.ShouldRun = false
		if output.WorkoutType == "" {
			output.WorkoutType = "rest"
		}
		if output.RiskLevel == "" {
			output.RiskLevel = "red"
		}
		if len(output.Explanation) < 2 {
			output.Explanation = []string{"安全优先，建议休息", "触发安全规则降级"}
		}
	}
	return RuleResult{Output: output, OverrideReason: override}
}
